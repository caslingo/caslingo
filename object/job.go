// Copyright 2026 The Caslingo Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package object

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/caslingo/caslingo/conf"
)

const (
	JobSync      = "sync"
	JobPush      = "push"
	JobTranslate = "translate"
	JobImport    = "import"

	JobRunning  = "running"
	JobDone     = "done"
	JobFailed   = "failed"
	JobCanceled = "canceled"
)

// Job is a sync, push or machine translation running in the background. A project runs one job at a time.
type Job struct {
	Id           string   `json:"id"`
	Project      string   `json:"project"`
	Kind         string   `json:"kind"`
	Languages    []string `json:"languages"`
	Author       string   `json:"author"`
	Status       string   `json:"status"`
	Progress     int      `json:"progress"`
	Total        int      `json:"total"`
	Message      string   `json:"message"`
	Url          string   `json:"url"`
	StartedTime  string   `json:"startedTime"`
	FinishedTime string   `json:"finishedTime"`

	options *TranslateOptions
	cancel  context.CancelFunc
}

// TranslateOptions says what a machine translation job does.
type TranslateOptions struct {
	Component string `json:"component"`
	File      string `json:"file"`
	// also redo the texts marked as needing an edit
	IncludeNeedsEdit bool `json:"includeNeedsEdit"`
	// the state machine translations get: StateTranslated, or StateNeedsEdit to have people review them
	State int `json:"state"`
	// only use the translation memory, without calling the model
	MemoryOnly bool `json:"memoryOnly"`
}

var (
	jobMutex    sync.Mutex
	runningJobs = map[string]*Job{}
	recentJobs  = map[string][]*Job{}
	pendingSync = map[string]bool{}
)

func (j *Job) snapshot() *Job {
	jobMutex.Lock()
	defer jobMutex.Unlock()
	copied := *j
	copied.Languages = append([]string{}, j.Languages...)
	return &copied
}

func (j *Job) setProgress(progress int, total int, message string) {
	jobMutex.Lock()
	defer jobMutex.Unlock()
	j.Progress = progress
	j.Total = total
	if message != "" {
		j.Message = message
	}
}

// GetJobs returns the project's running job (if any) first, then the recent ones, newest first.
func GetJobs(project string) []*Job {
	jobMutex.Lock()
	defer jobMutex.Unlock()

	res := []*Job{}
	if job := runningJobs[project]; job != nil {
		copied := *job
		res = append(res, &copied)
	}
	recent := recentJobs[project]
	for i := len(recent) - 1; i >= 0; i-- {
		copied := *recent[i]
		res = append(res, &copied)
	}
	return res
}

func CancelJob(project string) bool {
	jobMutex.Lock()
	defer jobMutex.Unlock()
	job := runningJobs[project]
	if job == nil {
		return false
	}
	job.cancel()
	return true
}

// StartJob starts a job unless the project is already running one. A sync asked for while another job runs
// (a webhook during a push) runs after it.
func StartJob(project *Project, kind string, languages []string, options *TranslateOptions, author string) (*Job, error) {
	jobMutex.Lock()
	if running := runningJobs[project.Name]; running != nil {
		if kind == JobSync {
			pendingSync[project.Name] = true
		}
		jobMutex.Unlock()
		return nil, fmt.Errorf("the project is busy with a %s job, try again when it is done", running.Kind)
	}

	ctx, cancel := context.WithCancel(context.Background())
	job := &Job{
		Id:          randomString(12),
		Project:     project.Name,
		Kind:        kind,
		Languages:   languages,
		Author:      author,
		Status:      JobRunning,
		StartedTime: getCurrentTime(),
		options:     options,
		cancel:      cancel,
	}
	runningJobs[project.Name] = job
	jobMutex.Unlock()

	go runJob(ctx, project.Name, job)
	return job.snapshot(), nil
}

func runJob(ctx context.Context, projectName string, job *Job) {
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal error: %v", r)
		}
		finishJob(ctx, projectName, job, err)
	}()

	project, err := GetProject(projectName)
	if err != nil {
		return
	}
	if project == nil {
		err = fmt.Errorf("the project %s does not exist", projectName)
		return
	}

	switch job.Kind {
	case JobSync:
		err = runSync(ctx, project, job)
	case JobPush:
		err = runPush(ctx, project, job)
	case JobTranslate:
		err = runTranslate(ctx, project, job, job.Languages, job.options)
	case JobImport:
		err = runImport(project, job)
	default:
		err = fmt.Errorf("unknown job: %s", job.Kind)
	}
}

func finishJob(ctx context.Context, projectName string, job *Job, err error) {
	jobMutex.Lock()
	job.FinishedTime = getCurrentTime()
	switch {
	case ctx.Err() != nil:
		job.Status = JobCanceled
		job.Message = "Canceled. " + job.Message
	case err != nil:
		job.Status = JobFailed
		job.Message = err.Error()
	default:
		job.Status = JobDone
	}
	job.cancel()
	delete(runningJobs, projectName)
	recent := append(recentJobs[projectName], job)
	if len(recent) > 20 {
		recent = recent[len(recent)-20:]
	}
	recentJobs[projectName] = recent
	runPending := pendingSync[projectName]
	delete(pendingSync, projectName)
	jobMutex.Unlock()

	if err != nil {
		logs.Warn("%s job of %s failed: %v", job.Kind, projectName, err)
	}
	project, _ := GetProject(projectName)
	if project != nil {
		if job.Kind == JobSync || job.Kind == JobPush {
			project.LastError = ""
			if err != nil {
				project.LastError = err.Error()
			}
			_ = updateProjectStatus(project, "last_error")
		}
		if runPending {
			_, _ = StartJob(project, JobSync, nil, nil, "")
		}
	}
}

// syncProject pulls the repository and reads every component.
func syncProject(project *Project, job *Job) (*SyncResult, error) {
	commit := ""
	if project.RepoUrl != "" {
		job.setProgress(0, 0, "Pulling the repository")
		var err error
		commit, err = pullRepo(project)
		if err != nil {
			return nil, err
		}
	}

	components, err := GetComponents(project.Name)
	if err != nil {
		return nil, err
	}
	result := &SyncResult{}
	for i, component := range components {
		job.setProgress(i, len(components), "Reading "+component.DisplayName)
		err = syncComponent(project, component, result)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", component.Name, err)
		}
	}

	project.LastSyncTime = getCurrentTime()
	project.LastSyncCommit = commit
	err = updateProjectStatus(project, "last_sync_time", "last_sync_commit")
	return result, err
}

func runSync(ctx context.Context, project *Project, job *Job) error {
	result, err := syncProject(project, job)
	if err != nil {
		return err
	}
	job.setProgress(1, 1, result.String())
	err = AddChange(&Change{Project: project.Name, Action: ActionSync, Author: job.Author, NewText: result.String()})
	if err != nil {
		return err
	}

	if project.AutoTranslate && IsAiEnabled() && result.Added+result.Changed > 0 {
		err = runTranslate(ctx, project, job, project.Languages, &TranslateOptions{State: StateTranslated})
		if err != nil {
			return err
		}
	}
	if project.AutoPush && project.PushMode != PushModeNone && result.Added+result.Changed+result.Removed > 0 {
		return pushProject(project, job)
	}
	job.setProgress(1, 1, result.String())
	return nil
}

func runPush(ctx context.Context, project *Project, job *Job) error {
	if project.PushMode == PushModeNone {
		return fmt.Errorf("pushing is turned off in the project's settings")
	}
	// pull first, so that translations edited in the repository since the last sync are not overwritten
	_, err := syncProject(project, job)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return nil
	}
	return pushProject(project, job)
}

func pushProject(project *Project, job *Job) error {
	job.setProgress(0, 1, "Writing the translation files")
	paths, err := writeTranslationsToRepo(project)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		job.setProgress(1, 1, "The repository already has every translation, nothing to push")
		return nil
	}

	job.setProgress(0, 1, fmt.Sprintf("Pushing %d files", len(paths)))
	pushed, pullRequestUrl, err := commitAndPush(project, paths)
	if err != nil {
		return err
	}
	message := fmt.Sprintf("Pushed %d files to %s", len(paths), project.Branch)
	if project.PushMode == PushModePullRequest {
		message = fmt.Sprintf("Pushed %d files to %s", len(paths), project.PushBranch)
		if pullRequestUrl != "" {
			message += ", pull request: " + pullRequestUrl
		}
	}
	if !pushed {
		message = "Nothing changed, nothing to push"
	}

	jobMutex.Lock()
	job.Url = pullRequestUrl
	jobMutex.Unlock()
	job.setProgress(1, 1, message)

	project.LastPushTime = getCurrentTime()
	err = updateProjectStatus(project, "last_push_time")
	if err != nil {
		return err
	}
	return AddChange(&Change{Project: project.Name, Action: ActionPush, Author: job.Author, NewText: message})
}

// RunPeriodicSync syncs every project with a repository every syncIntervalMinutes.
func RunPeriodicSync() {
	minutes := conf.GetConfigInt("syncIntervalMinutes", 0)
	if minutes <= 0 {
		return
	}
	for range time.Tick(time.Duration(minutes) * time.Minute) {
		projects, err := GetProjects()
		if err != nil {
			logs.Warn("periodic sync: %v", err)
			continue
		}
		for _, project := range projects {
			if project.RepoUrl != "" {
				_, _ = StartJob(project, JobSync, nil, nil, "")
			}
		}
	}
}
