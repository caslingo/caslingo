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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/beego/beego/v2/core/logs"
	"github.com/caslingo/caslingo/object"
)

func (c *ApiController) sendFile(name string, contentType string, data []byte) {
	c.EnableRender = false
	header := c.Ctx.ResponseWriter.Header()
	header.Set("Content-Type", contentType)
	header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, strings.ReplaceAll(name, `"`, "")))
	header.Set("Cache-Control", "no-store")
	c.Ctx.ResponseWriter.WriteHeader(http.StatusOK)
	_, _ = c.Ctx.ResponseWriter.Write(data)
}

// DownloadTranslations
// @Title DownloadTranslations
// @Description a zip of the project's translation files at their paths in the repository, for builds that take
// the translations from Caslingo: curl -H "Authorization: Bearer <access token>" ... > t.zip && unzip -o t.zip
// @Param project query string true "The project name"
// @Param component query string false "Only this component"
// @Param language query string false "Only this language"
// @router /download-translations [get]
func (c *ApiController) DownloadTranslations() {
	project := c.getProject(c.GetSessionUser())
	if project == nil {
		return
	}
	language := c.GetString("language")
	if language != "" && !containsString(project.Languages, language) {
		c.ResponseError("The project has no language " + language)
		return
	}

	data, err := object.ExportZip(project, c.GetString("component"), language)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.sendFile(project.Name+"-translations.zip", "application/zip", data)
}

// DownloadFile
// @Title DownloadFile
// @Description one translation file, as it would be pushed
// @Param project query string true "The project name"
// @Param component query string true "The component name"
// @Param file query string true "The source file"
// @Param language query string true "The language code"
// @router /download-file [get]
func (c *ApiController) DownloadFile() {
	project := c.getProject(c.GetSessionUser())
	if project == nil {
		return
	}
	language, ok := c.getLanguage(project)
	if !ok {
		return
	}
	component, err := object.GetComponent(project.Name, c.GetString("component"))
	if err != nil || component == nil {
		c.ResponseError("The component does not exist")
		return
	}

	file, err := object.ExportFile(project, component, c.GetString("file"), language)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	if file == nil {
		c.ResponseError("Nothing is translated in this file yet")
		return
	}
	c.sendFile(path.Base(file.Path), "application/octet-stream", file.Content)
}

// UploadSource
// @Title UploadSource
// @Description stores a source file generated at build time, for components whose source is uploaded, then syncs:
// curl -H "Authorization: Bearer <access token>" -H "Content-Type: application/octet-stream" --data-binary @i18n/en/code.json
// ".../api/upload-source?project=x&path=i18n/en/code.json" (without the content type curl sends a form, which beego rejects)
// @Param project query string true "The project name"
// @Param path query string true "The file's path, as the component's source pattern matches it"
// @Param sync query bool false "Start a sync afterwards (default true)"
// @router /upload-source [post]
func (c *ApiController) UploadSource() {
	user := c.GetSessionUser()
	project := c.getProject(user)
	if project == nil {
		return
	}
	if !project.IsMaintainer(user) && !c.hasProjectToken(project) {
		c.ResponseError("Only the project's maintainers can do this")
		return
	}

	data := c.Ctx.Input.RequestBody
	if file, _, err := c.GetFile("file"); err == nil {
		defer file.Close()
		data, err = io.ReadAll(file)
		if err != nil {
			c.ResponseError(err.Error())
			return
		}
	}
	if len(data) == 0 {
		c.ResponseError("The file is empty")
		return
	}

	err := object.SaveUploadedSource(project.Name, c.GetString("path"), data)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	if c.GetString("sync") != "false" {
		_, err = object.StartJob(project, object.JobSync, nil, nil, "")
		if err != nil {
			// a running job syncs again when it is done
			logs.Info("upload-source of %s: %v", project.Name, err)
		}
	}
	c.ResponseOk()
}

// ImportTranslations
// @Title ImportTranslations
// @Description takes the translations of a zip of translation files laid out as in the repository (an export from
// Crowdin or another platform); pull the project first. It runs as an import job:
// curl -H "Authorization: Bearer <access token>" -H "Content-Type: application/zip" --data-binary @export.zip ".../api/import-translations?project=x"
// @Param project query string true "The project name"
// @router /import-translations [post]
func (c *ApiController) ImportTranslations() {
	user := c.GetSessionUser()
	project := c.getMaintainedProject(user)
	if project == nil {
		return
	}

	data := c.Ctx.Input.RequestBody
	if file, _, err := c.GetFile("file"); err == nil {
		defer file.Close()
		data, err = io.ReadAll(file)
		if err != nil {
			c.ResponseError(err.Error())
			return
		}
	}
	err := object.SaveImportZip(project.Name, data)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	author := ""
	if user != nil {
		author = user.Name
	}
	job, err := object.StartJob(project, object.JobImport, nil, nil, author)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(job)
}

// Webhook
// @Title Webhook
// @Description GitHub push webhook (content type application/json, secret from the project's settings): syncs the
// project when its branch changes
// @Param project query string true "The project name"
// @router /webhook [post]
func (c *ApiController) Webhook() {
	project, err := object.GetProject(c.GetString("project"))
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	if project == nil {
		c.ResponseError("The project does not exist")
		return
	}

	body := c.Ctx.Input.RequestBody
	bearer, _ := c.getBearerProject()
	signed := object.VerifyGithubSignature(project.WebhookSecret, body, c.Ctx.Request.Header.Get("X-Hub-Signature-256"))
	if !signed && (bearer == nil || bearer.Name != project.Name) {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusUnauthorized)
		c.ResponseError("Invalid signature")
		return
	}

	event := c.Ctx.Request.Header.Get("X-GitHub-Event")
	if event == "ping" {
		c.ResponseOk("pong")
		return
	}
	if event == "push" {
		var push struct {
			Ref string `json:"ref"`
		}
		_ = json.Unmarshal(body, &push)
		if push.Ref != "refs/heads/"+project.Branch {
			c.ResponseOk("ignored: not " + project.Branch)
			return
		}
	}

	_, err = object.StartJob(project, object.JobSync, nil, nil, "")
	if err != nil {
		c.ResponseOk("queued after the running job")
		return
	}
	c.ResponseOk("sync started")
}
