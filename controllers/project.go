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

package controllers

import (
	"github.com/caslingo/caslingo/object"
)

// ProjectInfo is a project as the pages show it, with what the user may do in it.
type ProjectInfo struct {
	*object.Project
	Stats        []*object.Stats `json:"stats"`
	CanTranslate bool            `json:"canTranslate"`
	CanMaintain  bool            `json:"canMaintain"`
}

func newProjectInfo(project *object.Project, user *object.User, withStats bool) (*ProjectInfo, error) {
	info := &ProjectInfo{
		Project:      project,
		CanTranslate: project.IsTranslator(user),
		CanMaintain:  project.IsMaintainer(user),
	}
	if withStats {
		stats, err := object.GetProjectStats(project)
		if err != nil {
			return nil, err
		}
		info.Stats = stats
	}
	// the secrets come from get-project-secrets, for maintainers
	project.HideSecrets()
	return info, nil
}

// GetProjects
// @Title GetProjects
// @Description lists the projects the user can see, with their progress
// @router /get-projects [get]
func (c *ApiController) GetProjects() {
	user := c.GetSessionUser()
	projects, err := object.GetProjects()
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	res := []*ProjectInfo{}
	for _, project := range projects {
		if !project.CanSee(user) {
			continue
		}
		info, err := newProjectInfo(project, user, true)
		if err != nil {
			c.ResponseError(err.Error())
			return
		}
		res = append(res, info)
	}
	c.ResponseOk(res)
}

// GetProject
// @Title GetProject
// @Description returns a project with its components, progress and jobs
// @Param project query string true "The project name"
// @router /get-project [get]
func (c *ApiController) GetProject() {
	user := c.GetSessionUser()
	project := c.getProject(user)
	if project == nil {
		return
	}

	components, err := object.GetComponents(project.Name)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	info, err := newProjectInfo(project, user, true)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(info, map[string]interface{}{
		"components": components,
		"jobs":       object.GetJobs(project.Name),
	})
}

// GetProjectSecrets
// @Title GetProjectSecrets
// @Description returns the webhook secret and access token of a project, for its maintainers
// @Param project query string true "The project name"
// @router /get-project-secrets [get]
func (c *ApiController) GetProjectSecrets() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	project := c.getMaintainedProject(user)
	if project == nil {
		return
	}
	c.ResponseOk(map[string]string{"webhookSecret": project.WebhookSecret, "accessToken": project.AccessToken})
}

// AddProject
// @Title AddProject
// @Description creates a project (administrators only)
// @Param body body object.Project true "The project"
// @router /add-project [post]
func (c *ApiController) AddProject() {
	user := c.RequireAdmin()
	if user == nil {
		return
	}

	project := &object.Project{}
	if !c.readJsonBody(project) {
		return
	}
	if len(project.Maintainers) == 0 {
		project.Maintainers = []string{user.Name}
	}
	err := object.AddProject(project)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(project.Name)
}

// UpdateProject
// @Title UpdateProject
// @Description saves a project's settings (maintainers only)
// @Param project query string true "The project name"
// @Param body body object.Project true "The project"
// @router /update-project [post]
func (c *ApiController) UpdateProject() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	project := c.getMaintainedProject(user)
	if project == nil {
		return
	}

	updated := &object.Project{}
	if !c.readJsonBody(updated) {
		return
	}
	if !user.IsAdmin && !containsString(updated.Maintainers, user.Name) {
		// a maintainer cannot lock themselves out
		updated.Maintainers = append(updated.Maintainers, user.Name)
	}
	err := object.UpdateProject(project.Name, updated)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk()
}

// RegenerateProjectTokens
// @Title RegenerateProjectTokens
// @Description makes a new webhook secret and access token for a project, so the old ones stop working
// @Param project query string true "The project name"
// @router /regenerate-project-tokens [post]
func (c *ApiController) RegenerateProjectTokens() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	project := c.getMaintainedProject(user)
	if project == nil {
		return
	}
	err := object.RegenerateProjectTokens(project.Name)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk()
}

// DeleteProject
// @Title DeleteProject
// @Description deletes a project with all its strings and translations (administrators only)
// @Param project query string true "The project name"
// @router /delete-project [post]
func (c *ApiController) DeleteProject() {
	user := c.RequireAdmin()
	if user == nil {
		return
	}
	project := c.getProject(user)
	if project == nil {
		return
	}
	if object.CancelJob(project.Name) {
		c.ResponseError("A job is running in the project; it has been canceled, try again in a moment")
		return
	}
	err := object.DeleteProject(project.Name)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk()
}

// AddComponent
// @Title AddComponent
// @Description adds a component to a project (maintainers only)
// @Param project query string true "The project name"
// @Param body body object.Component true "The component"
// @router /add-component [post]
func (c *ApiController) AddComponent() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	project := c.getMaintainedProject(user)
	if project == nil {
		return
	}

	component := &object.Component{}
	if !c.readJsonBody(component) {
		return
	}
	component.Project = project.Name
	err := object.AddComponent(component)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(component.Name)
}

// UpdateComponent
// @Title UpdateComponent
// @Description saves a component (maintainers only); the files are read again on the next sync
// @Param project query string true "The project name"
// @Param component query string true "The component name"
// @Param body body object.Component true "The component"
// @router /update-component [post]
func (c *ApiController) UpdateComponent() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	project := c.getMaintainedProject(user)
	if project == nil {
		return
	}

	component := &object.Component{}
	if !c.readJsonBody(component) {
		return
	}
	err := object.UpdateComponent(project.Name, c.GetString("component"), component)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk()
}

// DeleteComponent
// @Title DeleteComponent
// @Description deletes a component with its strings and translations (maintainers only)
// @Param project query string true "The project name"
// @Param component query string true "The component name"
// @router /delete-component [post]
func (c *ApiController) DeleteComponent() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	project := c.getMaintainedProject(user)
	if project == nil {
		return
	}
	err := object.DeleteComponent(project.Name, c.GetString("component"))
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk()
}
