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

package main

import (
	"fmt"
	"net/http"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/caslingo/caslingo/conf"
	"github.com/caslingo/caslingo/object"
	"github.com/caslingo/caslingo/routers"
)

func main() {
	object.InitAdapter()

	web.BConfig.WebConfig.Session.SessionOn = true
	web.BConfig.WebConfig.Session.SessionName = "caslingo_session_id"
	web.BConfig.WebConfig.Session.SessionProvider = "file"
	web.BConfig.WebConfig.Session.SessionProviderConfig = "./tmp"
	web.BConfig.WebConfig.Session.SessionGCMaxLifetime = 3600 * 24 * 30
	web.BConfig.WebConfig.Session.SessionCookieLifeTime = 3600 * 24 * 30
	web.BConfig.WebConfig.Session.SessionCookieSameSite = http.SameSiteLaxMode
	web.BConfig.MaxUploadSize = 64 << 20

	routers.InitAPI()
	web.InsertFilter("*", web.BeforeRouter, routers.StaticFilter)

	logs.SetLogFuncCall(false)
	if object.IsAiEnabled() {
		logs.Info("machine translation is enabled, model: %s", conf.GetConfigString("aiModel"))
	} else {
		logs.Warn("machine translation is disabled: set aiBaseUrl, aiModel and aiApiKey")
	}

	go object.RunPeriodicSync()

	port := web.AppConfig.DefaultInt("httpport", 16500)
	web.Run(fmt.Sprintf(":%v", port))
}
