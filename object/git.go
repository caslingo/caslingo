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
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/caslingo/caslingo/conf"
)

func getReposDir() string {
	dir, err := filepath.Abs(conf.GetConfigString("reposDir"))
	if err != nil {
		return conf.GetConfigString("reposDir")
	}
	return dir
}

// getAuthUrl puts the project's token into an https repository URL, the way GitHub and GitLab take it.
func getAuthUrl(project *Project) string {
	if project.RepoToken == "" {
		return project.RepoUrl
	}
	u, err := url.Parse(project.RepoUrl)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return project.RepoUrl
	}
	u.User = url.UserPassword("x-access-token", project.RepoToken)
	return u.String()
}

// runGit runs git in dir, hiding the token from what it returns.
func runGit(project *Project, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=echo")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()

	output := strings.TrimSpace(stdout.String())
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if project.RepoToken != "" {
			message = strings.ReplaceAll(message, project.RepoToken, "***")
		}
		return output, fmt.Errorf("git %s: %v: %s", args[0], err, message)
	}
	return output, nil
}

// pullRepo clones the repository the first time and later moves it to the branch's latest commit,
// throwing away what an earlier push left (the database has it all). It returns the commit.
func pullRepo(project *Project) (string, error) {
	if project.RepoUrl == "" {
		return "", fmt.Errorf("the project has no repository URL")
	}

	dir := project.GetRepoDir()
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		err = os.MkdirAll(getReposDir(), 0o755)
		if err != nil {
			return "", err
		}
		_ = os.RemoveAll(dir)
		_, err = runGit(project, getReposDir(), "clone", "--depth", "1", "--single-branch", "--branch", project.Branch, getAuthUrl(project), project.Name)
		if err != nil {
			return "", err
		}
	} else {
		_, err = runGit(project, dir, "remote", "set-url", "origin", getAuthUrl(project))
		if err != nil {
			return "", err
		}
		_, err = runGit(project, dir, "fetch", "--depth", "1", "origin", project.Branch)
		if err != nil {
			return "", err
		}
		_, err = runGit(project, dir, "checkout", "-f", "-B", project.Branch, "FETCH_HEAD")
		if err != nil {
			return "", err
		}
		_, err = runGit(project, dir, "clean", "-fdx")
		if err != nil {
			return "", err
		}
	}

	return runGit(project, dir, "rev-parse", "HEAD")
}

// commitAndPush commits what changed in the working tree and pushes it: to the branch itself, or to the push branch
// with a pull request. It returns false when nothing changed.
func commitAndPush(project *Project, paths []string) (bool, string, error) {
	dir := project.GetRepoDir()
	args := append([]string{"add", "--"}, paths...)
	_, err := runGit(project, dir, args...)
	if err != nil {
		return false, "", err
	}

	status, err := runGit(project, dir, "status", "--porcelain")
	if err != nil {
		return false, "", err
	}
	if status == "" {
		return false, "", nil
	}

	_, err = runGit(project, dir, "-c", "user.name="+project.CommitName, "-c", "user.email="+project.CommitEmail, "commit", "-m", project.CommitMessage)
	if err != nil {
		return false, "", err
	}

	if project.PushMode == PushModePullRequest {
		_, err = runGit(project, dir, "push", "--force", "origin", "HEAD:refs/heads/"+project.PushBranch)
		if err != nil {
			return false, "", err
		}
		pullRequestUrl, err := ensurePullRequest(project)
		return true, pullRequestUrl, err
	}

	_, err = runGit(project, dir, "push", "origin", "HEAD:refs/heads/"+project.Branch)
	return true, "", err
}
