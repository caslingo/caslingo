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
	"net"
	"strings"

	"github.com/beego/beego/v2/server/web"
	"github.com/caslingo/caslingo/conf"
	"github.com/caslingo/caslingo/object"
)

type ApiController struct {
	web.Controller
}

type Response struct {
	Status string      `json:"status"`
	Msg    string      `json:"msg"`
	Data   interface{} `json:"data"`
	Data2  interface{} `json:"data2"`
}

func (c *ApiController) ResponseOk(data ...interface{}) {
	res := &Response{Status: "ok"}
	if len(data) > 0 {
		res.Data = data[0]
	}
	if len(data) > 1 {
		res.Data2 = data[1]
	}
	c.Data["json"] = res
	c.ServeJSON()
}

func (c *ApiController) ResponseError(error string) {
	c.Data["json"] = &Response{Status: "error", Msg: error}
	c.ServeJSON()
}

func (c *ApiController) getClientIp() string {
	if conf.GetConfigBool("trustProxyHeader") {
		forwardedFor := c.Ctx.Request.Header.Get("X-Forwarded-For")
		if forwardedFor != "" {
			return strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
		}
		realIp := c.Ctx.Request.Header.Get("X-Real-IP")
		if realIp != "" {
			return realIp
		}
	}

	host, _, err := net.SplitHostPort(c.Ctx.Request.RemoteAddr)
	if err != nil {
		return c.Ctx.Request.RemoteAddr
	}
	return host
}

func (c *ApiController) readJsonBody(v interface{}) bool {
	err := json.Unmarshal(c.Ctx.Input.RequestBody, v)
	if err != nil {
		c.ResponseError(fmt.Sprintf("invalid request body: %v", err))
		return false
	}
	return true
}

// GetSessionUser returns the signed-in user, or nil. The session stores the user as JSON so any session provider can keep it.
func (c *ApiController) GetSessionUser() *object.User {
	data, ok := c.GetSession("user").(string)
	if !ok || data == "" {
		return nil
	}

	user := &object.User{}
	err := json.Unmarshal([]byte(data), user)
	if err != nil {
		return nil
	}
	user.IsAdmin = user.IsAdmin || object.IsAdminUser(user.Name)
	return user
}

func (c *ApiController) SetSessionUser(user *object.User) error {
	if user == nil {
		return c.DelSession("user")
	}

	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return c.SetSession("user", string(data))
}

// RequireSignedInUser responds with an error and returns nil when nobody is signed in.
func (c *ApiController) RequireSignedInUser() *object.User {
	user := c.GetSessionUser()
	if user == nil {
		c.ResponseError("Please sign in first")
		return nil
	}
	return user
}

func (c *ApiController) RequireAdmin() *object.User {
	user := c.RequireSignedInUser()
	if user == nil {
		return nil
	}
	if !user.IsAdmin {
		c.ResponseError("Only administrators can do this")
		return nil
	}
	return user
}

// getBearerProject is the project whose access token the request carries (for CI jobs), or nil.
func (c *ApiController) getBearerProject() (*object.Project, error) {
	auth := c.Ctx.Request.Header.Get("Authorization")
	token, ok := strings.CutPrefix(auth, "Bearer ")
	if !ok {
		token = c.GetString("token")
	}
	return object.GetProjectByAccessToken(strings.TrimSpace(token))
}

// getProject loads the project named in the request and checks the user may see it.
// It responds with the error and returns nil when not.
func (c *ApiController) getProject(user *object.User) *object.Project {
	name := c.GetString("project")
	project, err := object.GetProject(name)
	if err != nil {
		c.ResponseError(err.Error())
		return nil
	}

	bearer, err := c.getBearerProject()
	if err != nil {
		c.ResponseError(err.Error())
		return nil
	}
	if project == nil || (!project.CanSee(user) && (bearer == nil || bearer.Name != project.Name)) {
		c.ResponseError(fmt.Sprintf("The project %s does not exist", name))
		return nil
	}
	return project
}

// hasProjectToken tells whether the request carries the project's access token, which lets CI jobs and scripts
// do what the project's maintainers can.
func (c *ApiController) hasProjectToken(project *object.Project) bool {
	bearer, err := c.getBearerProject()
	return err == nil && bearer != nil && bearer.Name == project.Name
}

// getMaintainedProject loads the project and checks the user maintains it, or the request has its access token.
func (c *ApiController) getMaintainedProject(user *object.User) *object.Project {
	project := c.getProject(user)
	if project == nil {
		return nil
	}
	if !project.IsMaintainer(user) && !c.hasProjectToken(project) {
		c.ResponseError("Only the project's maintainers can do this")
		return nil
	}
	return project
}

func (c *ApiController) getLanguage(project *object.Project) (string, bool) {
	language := c.GetString("language")
	if !containsString(project.Languages, language) {
		c.ResponseError(fmt.Sprintf("The project has no language %s", language))
		return "", false
	}
	return language, true
}

func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
