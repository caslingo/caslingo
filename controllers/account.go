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
	"net"

	"github.com/beego/beego/v2/core/logs"
	"github.com/caslingo/caslingo/conf"
	"github.com/caslingo/caslingo/format"
	"github.com/caslingo/caslingo/object"
)

// Signin
// @Title Signin
// @Description signs in with the code Casdoor sends to /callback
// @Param code query string true "OAuth code"
// @Success 200 {object} object.User The signed-in user
// @router /signin [post]
func (c *ApiController) Signin() {
	code := c.GetString("code")
	if code == "" {
		c.ResponseError("Missing the sign-in code")
		return
	}

	user, err := object.GetUserByOAuthCode(code)
	if err != nil {
		logs.Warn("signin failed: %v", err)
		c.ResponseError("Sign-in failed: " + err.Error())
		return
	}

	err = c.SetSessionUser(user)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	logs.Info("signin: %s/%s from %s", user.Owner, user.Name, c.getClientIp())
	c.ResponseOk(user)
}

// DevSignin
// @Title DevSignin
// @Description signs in as "admin" without Casdoor, when devSignin = true and the request comes from this computer
// @router /dev-signin [post]
func (c *ApiController) DevSignin() {
	// the address of the connection itself, not a forwarded header anyone can set; and nothing forwarded at all,
	// since a reverse proxy on this computer (CasWAF, Nginx) connects from 127.0.0.1 too
	host, _, _ := net.SplitHostPort(c.Ctx.Request.RemoteAddr)
	forwarded := c.Ctx.Request.Header.Get("X-Forwarded-For") != "" || c.Ctx.Request.Header.Get("X-Real-IP") != ""
	if !conf.GetConfigBool("devSignin") || (host != "127.0.0.1" && host != "::1") || forwarded {
		c.ResponseError("Sign-in without Casdoor is turned off")
		return
	}

	user := &object.User{Owner: "built-in", Name: "admin", DisplayName: "Admin", IsAdmin: true}
	err := c.SetSessionUser(user)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk(user)
}

// Signout
// @Title Signout
// @Description signs out
// @router /signout [post]
func (c *ApiController) Signout() {
	err := c.SetSessionUser(nil)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk()
}

// GetAccount
// @Title GetAccount
// @Description returns the signed-in user, or null
// @Success 200 {object} object.User The signed-in user
// @router /get-account [get]
func (c *ApiController) GetAccount() {
	c.ResponseOk(c.GetSessionUser())
}

// GetConfig
// @Title GetConfig
// @Description returns what the pages need before sign-in: the site's name and logo, Casdoor, the languages and formats
// @router /get-config [get]
func (c *ApiController) GetConfig() {
	c.ResponseOk(map[string]interface{}{
		"siteName":   conf.GetConfigString("siteName"),
		"logoUrl":    conf.GetConfigString("logoUrl"),
		"faviconUrl": conf.GetConfigString("faviconUrl"),
		"footerHtml": conf.GetConfigString("footerHtml"),
		"casdoor":    object.GetCasdoorConfig(),
		"devSignin":  conf.GetConfigBool("devSignin"),
		"aiEnabled":  object.IsAiEnabled(),
		"aiModel":    conf.GetConfigString("aiModel"),
		"languages":  object.Languages,
		"formats":    format.Formats,
	})
}
