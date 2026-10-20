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
	"context"
	"time"

	"github.com/caslingo/caslingo/object"
)

// GetUnits
// @Title GetUnits
// @Description returns a page of a project's strings with their translations in a language
// @Param project query string true "The project name"
// @Param language query string true "The language code"
// @Param component query string false "Only this component"
// @Param file query string false "Only this source file"
// @Param filter query string false "all, untranslated, needs-edit, translated, unapproved, approved, checks or suggestions"
// @Param search query string false "Text in the source, the key or the translation"
// @Param page query int false "The page, from 1"
// @Param pageSize query int false "Strings per page, at most 200"
// @router /get-units [get]
func (c *ApiController) GetUnits() {
	user := c.GetSessionUser()
	project := c.getProject(user)
	if project == nil {
		return
	}
	language, ok := c.getLanguage(project)
	if !ok {
		return
	}

	page, _ := c.GetInt("page", 1)
	pageSize, _ := c.GetInt("pageSize", 50)
	units, total, err := object.GetEditorUnits(&object.UnitQuery{
		Project:   project.Name,
		Language:  language,
		Component: c.GetString("component"),
		File:      c.GetString("file"),
		Filter:    c.GetString("filter"),
		Search:    c.GetString("search"),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(units, total)
}

// GetFiles
// @Title GetFiles
// @Description lists the source files of a project or component
// @Param project query string true "The project name"
// @Param component query string false "Only this component"
// @router /get-files [get]
func (c *ApiController) GetFiles() {
	user := c.GetSessionUser()
	project := c.getProject(user)
	if project == nil {
		return
	}
	files, err := object.GetFiles(project.Name, c.GetString("component"))
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(files)
}

// getUnit loads the unit in the "id" parameter, which must belong to the project.
func (c *ApiController) getUnit(project *object.Project) *object.Unit {
	id, _ := c.GetInt64("id")
	unit, err := object.GetUnit(id)
	if err != nil {
		c.ResponseError(err.Error())
		return nil
	}
	if unit == nil || unit.Project != project.Name {
		c.ResponseError("The string does not exist")
		return nil
	}
	return unit
}

// GetUnitHelp
// @Title GetUnitHelp
// @Description returns what helps translate a string: translation memory matches, glossary terms, the other languages and its history
// @Param project query string true "The project name"
// @Param language query string true "The language code"
// @Param id query int true "The string id"
// @router /get-unit-help [get]
func (c *ApiController) GetUnitHelp() {
	user := c.GetSessionUser()
	project := c.getProject(user)
	if project == nil {
		return
	}
	language, ok := c.getLanguage(project)
	if !ok {
		return
	}
	unit := c.getUnit(project)
	if unit == nil {
		return
	}

	memory, err := object.GetMemoryMatches(unit, language)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	terms, err := object.GetMatchingTerms(project.Name, language, unit.Source)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	others, err := object.GetOtherTranslations(unit.Id, language)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	history, err := object.GetUnitChanges(unit.Id, language)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(map[string]interface{}{
		"memory":  memory,
		"terms":   terms,
		"others":  others,
		"history": history,
	})
}

type saveTranslationBody struct {
	Text  string `json:"text"`
	State int    `json:"state"`
}

// SaveTranslation
// @Title SaveTranslation
// @Description saves a string's translation; for someone who cannot translate the project it becomes a suggestion
// @Param project query string true "The project name"
// @Param language query string true "The language code"
// @Param id query int true "The string id"
// @Param body body saveTranslationBody true "The text and the state (10 needs edit, 20 translated, 30 approved)"
// @router /save-translation [post]
func (c *ApiController) SaveTranslation() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	project := c.getProject(user)
	if project == nil {
		return
	}
	language, ok := c.getLanguage(project)
	if !ok {
		return
	}
	unit := c.getUnit(project)
	if unit == nil {
		return
	}
	body := &saveTranslationBody{}
	if !c.readJsonBody(body) {
		return
	}

	if !project.IsTranslator(user) {
		if body.Text == "" {
			c.ResponseError("The suggestion is empty")
			return
		}
		suggestion, err := object.AddSuggestion(unit, language, body.Text, user.Name)
		if err != nil {
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(nil, suggestion)
		return
	}

	if body.State == object.StateApproved && !project.IsMaintainer(user) {
		body.State = object.StateTranslated
	}
	translation, err := object.SaveTranslation(unit, language, body.Text, body.State, "human", user.Name)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(translation)
}

// AcceptSuggestion
// @Title AcceptSuggestion
// @Description makes a suggestion the string's translation and removes it
// @Param project query string true "The project name"
// @Param id query int true "The suggestion id"
// @router /accept-suggestion [post]
func (c *ApiController) AcceptSuggestion() {
	c.handleSuggestion(true)
}

// DeleteSuggestion
// @Title DeleteSuggestion
// @Description removes a suggestion (translators, or the person who made it)
// @Param project query string true "The project name"
// @Param id query int true "The suggestion id"
// @router /delete-suggestion [post]
func (c *ApiController) DeleteSuggestion() {
	c.handleSuggestion(false)
}

func (c *ApiController) handleSuggestion(accept bool) {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	project := c.getProject(user)
	if project == nil {
		return
	}
	id, _ := c.GetInt64("id")
	suggestion, err := object.GetSuggestion(id)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	if suggestion == nil || suggestion.Project != project.Name {
		c.ResponseError("The suggestion does not exist")
		return
	}
	if !project.IsTranslator(user) && (accept || suggestion.Author != user.Name) {
		c.ResponseError("Only the project's translators can do this")
		return
	}

	var translation *object.Translation
	if accept {
		unit, err := object.GetUnit(suggestion.UnitId)
		if err != nil || unit == nil {
			c.ResponseError("The string does not exist")
			return
		}
		translation, err = object.SaveTranslation(unit, suggestion.Language, suggestion.Text, object.StateTranslated, "suggestion", user.Name)
		if err != nil {
			c.ResponseError(err.Error())
			return
		}
	}
	err = object.DeleteSuggestion(id)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(translation)
}

// MachineTranslateUnit
// @Title MachineTranslateUnit
// @Description asks the model for a translation of one string, without saving it
// @Param project query string true "The project name"
// @Param language query string true "The language code"
// @Param id query int true "The string id"
// @router /machine-translate-unit [post]
func (c *ApiController) MachineTranslateUnit() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	project := c.getProject(user)
	if project == nil {
		return
	}
	if !project.IsTranslator(user) {
		c.ResponseError("Only the project's translators can do this")
		return
	}
	language, ok := c.getLanguage(project)
	if !ok {
		return
	}
	unit := c.getUnit(project)
	if unit == nil {
		return
	}

	terms, err := object.GetTerms(project.Name, language)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	texts, err := object.MachineTranslate(ctx, project, language, []*object.Unit{unit}, terms)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	text, ok := texts[unit.Id]
	if !ok {
		c.ResponseError("The model gave no translation")
		return
	}
	c.ResponseOk(text)
}
