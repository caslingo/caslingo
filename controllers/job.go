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
	"strings"

	"github.com/caslingo/caslingo/object"
)

// StartJob
// @Title StartJob
// @Description starts a sync (pull the repository), a push (commit the translations) or a machine translation
// @Param project query string true "The project name"
// @Param kind query string true "sync, push or translate"
// @Param languages query string false "For translate: comma-separated language codes, empty for all"
// @Param body body object.TranslateOptions false "For translate"
// @router /start-job [post]
func (c *ApiController) StartJob() {
	user := c.GetSessionUser()
	project := c.getProject(user)
	if project == nil {
		return
	}

	bearer, err := c.getBearerProject()
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	author := ""
	if user != nil {
		author = user.Name
	}
	if !project.IsMaintainer(user) && (bearer == nil || bearer.Name != project.Name) {
		c.ResponseError("Only the project's maintainers can do this")
		return
	}

	kind := c.GetString("kind")
	languages := []string{}
	for _, language := range strings.Split(c.GetString("languages"), ",") {
		if language = strings.TrimSpace(language); language != "" {
			languages = append(languages, language)
		}
	}
	options := &object.TranslateOptions{}
	if kind == object.JobTranslate && len(c.Ctx.Input.RequestBody) > 0 && !c.readJsonBody(options) {
		return
	}
	if kind != object.JobSync && kind != object.JobPush && kind != object.JobTranslate {
		c.ResponseError("Unknown job: " + kind)
		return
	}

	job, err := object.StartJob(project, kind, languages, options, author)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(job)
}

// GetJobs
// @Title GetJobs
// @Description returns the project's running job and the recent ones
// @Param project query string true "The project name"
// @router /get-jobs [get]
func (c *ApiController) GetJobs() {
	project := c.getProject(c.GetSessionUser())
	if project == nil {
		return
	}
	c.ResponseOk(object.GetJobs(project.Name))
}

// CancelJob
// @Title CancelJob
// @Description stops the project's running job
// @Param project query string true "The project name"
// @router /cancel-job [post]
func (c *ApiController) CancelJob() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	project := c.getMaintainedProject(user)
	if project == nil {
		return
	}
	object.CancelJob(project.Name)
	c.ResponseOk()
}

// GetChanges
// @Title GetChanges
// @Description returns a page of the project's history, newest first
// @Param project query string true "The project name"
// @Param language query string false "Only this language"
// @Param author query string false "Only this person"
// @Param page query int false "The page, from 1"
// @router /get-changes [get]
func (c *ApiController) GetChanges() {
	project := c.getProject(c.GetSessionUser())
	if project == nil {
		return
	}
	page, _ := c.GetInt("page", 1)
	pageSize, _ := c.GetInt("pageSize", 50)
	changes, total, err := object.GetChanges(project.Name, c.GetString("language"), c.GetString("author"), page, pageSize)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	contributors, err := object.GetContributors(project.Name)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(changes, map[string]interface{}{"total": total, "contributors": contributors})
}

// GetTerms
// @Title GetTerms
// @Description returns the project's glossary
// @Param project query string true "The project name"
// @Param language query string false "Only this language"
// @router /get-terms [get]
func (c *ApiController) GetTerms() {
	project := c.getProject(c.GetSessionUser())
	if project == nil {
		return
	}
	terms, err := object.GetTerms(project.Name, c.GetString("language"))
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(terms)
}

func (c *ApiController) getTranslatedProject() *object.Project {
	user := c.RequireSignedInUser()
	if user == nil {
		return nil
	}
	project := c.getProject(user)
	if project == nil {
		return nil
	}
	if !project.IsTranslator(user) {
		c.ResponseError("Only the project's translators can do this")
		return nil
	}
	return project
}

// AddTerm
// @Title AddTerm
// @Description adds a glossary entry (translators)
// @Param project query string true "The project name"
// @Param body body object.Term true "The entry"
// @router /add-term [post]
func (c *ApiController) AddTerm() {
	project := c.getTranslatedProject()
	if project == nil {
		return
	}
	term := &object.Term{}
	if !c.readJsonBody(term) {
		return
	}
	if !containsString(project.Languages, term.Language) {
		c.ResponseError("The project has no language " + term.Language)
		return
	}
	term.Project = project.Name
	err := object.AddTerm(term)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(term)
}

// UpdateTerm
// @Title UpdateTerm
// @Description changes a glossary entry (translators)
// @Param project query string true "The project name"
// @Param id query int true "The entry id"
// @Param body body object.Term true "The entry"
// @router /update-term [post]
func (c *ApiController) UpdateTerm() {
	c.handleTerm(false)
}

// DeleteTerm
// @Title DeleteTerm
// @Description removes a glossary entry (translators)
// @Param project query string true "The project name"
// @Param id query int true "The entry id"
// @router /delete-term [post]
func (c *ApiController) DeleteTerm() {
	c.handleTerm(true)
}

func (c *ApiController) handleTerm(remove bool) {
	project := c.getTranslatedProject()
	if project == nil {
		return
	}
	id, _ := c.GetInt64("id")
	term, err := object.GetTerm(id)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	if term == nil || term.Project != project.Name {
		c.ResponseError("The glossary entry does not exist")
		return
	}

	if remove {
		err = object.DeleteTerm(id)
	} else {
		updated := &object.Term{}
		if !c.readJsonBody(updated) {
			return
		}
		if !containsString(project.Languages, updated.Language) {
			c.ResponseError("The project has no language " + updated.Language)
			return
		}
		err = object.UpdateTerm(id, updated)
	}
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk()
}
