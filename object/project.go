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
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	PushModeNone        = ""
	PushModeDirect      = "direct"
	PushModePullRequest = "pull-request"
)

// Project is one product to translate, usually one git repository.
type Project struct {
	Name        string `xorm:"varchar(100) notnull pk" json:"name"`
	DisplayName string `xorm:"varchar(200)" json:"displayName"`
	Description string `xorm:"varchar(1000)" json:"description"`
	WebsiteUrl  string `xorm:"varchar(500)" json:"websiteUrl"`
	CreatedTime string `xorm:"varchar(100)" json:"createdTime"`

	SourceLanguage string   `xorm:"varchar(20)" json:"sourceLanguage"`
	Languages      []string `xorm:"mediumtext" json:"languages"`

	// who can see and change it
	IsPublic    bool     `json:"isPublic"`
	OpenEditing bool     `json:"openEditing"`
	Maintainers []string `xorm:"mediumtext" json:"maintainers"`
	Translators []string `xorm:"mediumtext" json:"translators"`

	// the git repository
	RepoUrl       string `xorm:"varchar(500)" json:"repoUrl"`
	Branch        string `xorm:"varchar(100)" json:"branch"`
	RepoToken     string `xorm:"varchar(500)" json:"repoToken"`
	PushMode      string `xorm:"varchar(20)" json:"pushMode"`
	PushBranch    string `xorm:"varchar(100)" json:"pushBranch"`
	CommitMessage string `xorm:"varchar(500)" json:"commitMessage"`
	CommitName    string `xorm:"varchar(100)" json:"commitName"`
	CommitEmail   string `xorm:"varchar(100)" json:"commitEmail"`
	WebhookSecret string `xorm:"varchar(100)" json:"webhookSecret"`
	AccessToken   string `xorm:"varchar(100)" json:"accessToken"`

	// machine translation
	Instructions  string `xorm:"mediumtext" json:"instructions"`
	AutoTranslate bool   `json:"autoTranslate"`
	AutoPush      bool   `json:"autoPush"`

	LastSyncTime   string `xorm:"varchar(100)" json:"lastSyncTime"`
	LastSyncCommit string `xorm:"varchar(100)" json:"lastSyncCommit"`
	LastPushTime   string `xorm:"varchar(100)" json:"lastPushTime"`
	LastError      string `xorm:"mediumtext" json:"lastError"`
}

// secrets are shown only to the project's maintainers
func (p *Project) HideSecrets() {
	if p.RepoToken != "" {
		p.RepoToken = "***"
	}
	p.WebhookSecret = ""
	p.AccessToken = ""
}

func (p *Project) IsMaintainer(user *User) bool {
	return user != nil && (user.IsAdmin || containsString(p.Maintainers, user.Name))
}

func (p *Project) IsTranslator(user *User) bool {
	return user != nil && (p.OpenEditing || p.IsMaintainer(user) || containsString(p.Translators, user.Name))
}

func (p *Project) CanSee(user *User) bool {
	return p.IsPublic || p.IsTranslator(user) || (user != nil && p.IsMaintainer(user))
}

func (p *Project) GetRepoDir() string {
	return filepath.Join(getReposDir(), p.Name)
}

// fillLists makes the lists [] rather than null, for the pages
func (p *Project) fillLists() *Project {
	if p.Languages == nil {
		p.Languages = []string{}
	}
	if p.Maintainers == nil {
		p.Maintainers = []string{}
	}
	if p.Translators == nil {
		p.Translators = []string{}
	}
	return p
}

func GetProjects() ([]*Project, error) {
	projects := []*Project{}
	err := ormer.Engine.Asc("name").Find(&projects)
	for _, project := range projects {
		project.fillLists()
	}
	return projects, err
}

func GetProject(name string) (*Project, error) {
	project := &Project{Name: name}
	existed, err := ormer.Engine.Get(project)
	if err != nil || !existed {
		return nil, err
	}
	return project.fillLists(), nil
}

func GetProjectByAccessToken(token string) (*Project, error) {
	if token == "" {
		return nil, nil
	}
	project := &Project{AccessToken: token}
	existed, err := ormer.Engine.Get(project)
	if err != nil || !existed {
		return nil, err
	}
	return project.fillLists(), nil
}

func normalizeProject(p *Project) error {
	p.Name = strings.TrimSpace(p.Name)
	if !IsValidName(p.Name) {
		return fmt.Errorf("the project name may only have lowercase letters, digits, - and _: %s", p.Name)
	}
	if p.DisplayName == "" {
		p.DisplayName = p.Name
	}
	if p.SourceLanguage == "" {
		p.SourceLanguage = "en"
	}

	languages := []string{}
	for _, language := range p.Languages {
		language = strings.TrimSpace(language)
		if language != "" && language != p.SourceLanguage && !containsString(languages, language) {
			languages = append(languages, language)
		}
	}
	p.Languages = languages
	p.fillLists()

	if p.Branch == "" {
		p.Branch = "master"
	}
	if p.PushBranch == "" {
		p.PushBranch = "caslingo"
	}
	if p.CommitMessage == "" {
		p.CommitMessage = "feat: update translations"
	}
	if p.CommitName == "" {
		p.CommitName = "Caslingo"
	}
	if p.CommitEmail == "" {
		p.CommitEmail = "bot@caslingo.com"
	}
	if p.PushMode != PushModeDirect && p.PushMode != PushModePullRequest {
		p.PushMode = PushModeNone
	}
	return nil
}

func AddProject(project *Project) error {
	err := normalizeProject(project)
	if err != nil {
		return err
	}

	existing, err := GetProject(project.Name)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("the project %s already exists", project.Name)
	}

	project.CreatedTime = getCurrentTime()
	project.WebhookSecret = randomString(32)
	project.AccessToken = randomString(40)
	_, err = ormer.Engine.Insert(project)
	return err
}

// UpdateProject saves the settings; a RepoToken of "***" keeps the saved one.
func UpdateProject(name string, project *Project) error {
	existing, err := GetProject(name)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("the project %s does not exist", name)
	}

	project.Name = name
	err = normalizeProject(project)
	if err != nil {
		return err
	}
	if project.RepoToken == "***" {
		project.RepoToken = existing.RepoToken
	}
	project.CreatedTime = existing.CreatedTime
	project.WebhookSecret = existing.WebhookSecret
	project.AccessToken = existing.AccessToken
	project.LastSyncTime = existing.LastSyncTime
	project.LastSyncCommit = existing.LastSyncCommit
	project.LastPushTime = existing.LastPushTime
	project.LastError = existing.LastError

	if existing.RepoUrl != project.RepoUrl || existing.Branch != project.Branch {
		// a different repository: clone it again on the next sync
		_ = os.RemoveAll(existing.GetRepoDir())
		project.LastSyncCommit = ""
	}

	_, err = ormer.Engine.ID(name).AllCols().Update(project)
	return err
}

func RegenerateProjectTokens(name string) error {
	_, err := ormer.Engine.ID(name).Cols("webhook_secret", "access_token").Update(&Project{WebhookSecret: randomString(32), AccessToken: randomString(40)})
	return err
}

func updateProjectStatus(project *Project, cols ...string) error {
	_, err := ormer.Engine.ID(project.Name).Cols(cols...).Update(project)
	return err
}

func DeleteProject(name string) error {
	session := ormer.Engine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return err
	}

	for _, table := range []interface{}{new(Translation), new(Suggestion), new(Change), new(Unit), new(Term)} {
		_, err = session.Where("project = ?", name).Delete(table)
		if err != nil {
			_ = session.Rollback()
			return err
		}
	}
	_, err = session.Where("project = ?", name).Delete(new(Component))
	if err != nil {
		_ = session.Rollback()
		return err
	}
	_, err = session.ID(name).Delete(new(Project))
	if err != nil {
		_ = session.Rollback()
		return err
	}
	err = session.Commit()
	if err != nil {
		return err
	}

	_ = os.RemoveAll(filepath.Join(getReposDir(), name))
	_ = os.RemoveAll(getUploadDir(name))
	return nil
}
