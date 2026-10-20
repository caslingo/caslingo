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

package routers

import (
	"github.com/beego/beego/v2/server/web"
	"github.com/caslingo/caslingo/controllers"
)

func InitAPI() {
	web.Router("/api/get-config", &controllers.ApiController{}, "GET:GetConfig")
	web.Router("/api/signin", &controllers.ApiController{}, "POST:Signin")
	web.Router("/api/dev-signin", &controllers.ApiController{}, "POST:DevSignin")
	web.Router("/api/signout", &controllers.ApiController{}, "POST:Signout")
	web.Router("/api/get-account", &controllers.ApiController{}, "GET:GetAccount")

	web.Router("/api/get-projects", &controllers.ApiController{}, "GET:GetProjects")
	web.Router("/api/get-project", &controllers.ApiController{}, "GET:GetProject")
	web.Router("/api/get-project-secrets", &controllers.ApiController{}, "GET:GetProjectSecrets")
	web.Router("/api/add-project", &controllers.ApiController{}, "POST:AddProject")
	web.Router("/api/update-project", &controllers.ApiController{}, "POST:UpdateProject")
	web.Router("/api/regenerate-project-tokens", &controllers.ApiController{}, "POST:RegenerateProjectTokens")
	web.Router("/api/delete-project", &controllers.ApiController{}, "POST:DeleteProject")

	web.Router("/api/add-component", &controllers.ApiController{}, "POST:AddComponent")
	web.Router("/api/update-component", &controllers.ApiController{}, "POST:UpdateComponent")
	web.Router("/api/delete-component", &controllers.ApiController{}, "POST:DeleteComponent")

	web.Router("/api/get-units", &controllers.ApiController{}, "GET:GetUnits")
	web.Router("/api/get-files", &controllers.ApiController{}, "GET:GetFiles")
	web.Router("/api/get-unit-help", &controllers.ApiController{}, "GET:GetUnitHelp")
	web.Router("/api/save-translation", &controllers.ApiController{}, "POST:SaveTranslation")
	web.Router("/api/accept-suggestion", &controllers.ApiController{}, "POST:AcceptSuggestion")
	web.Router("/api/delete-suggestion", &controllers.ApiController{}, "POST:DeleteSuggestion")
	web.Router("/api/machine-translate-unit", &controllers.ApiController{}, "POST:MachineTranslateUnit")

	web.Router("/api/start-job", &controllers.ApiController{}, "POST:StartJob")
	web.Router("/api/get-jobs", &controllers.ApiController{}, "GET:GetJobs")
	web.Router("/api/cancel-job", &controllers.ApiController{}, "POST:CancelJob")
	web.Router("/api/get-changes", &controllers.ApiController{}, "GET:GetChanges")

	web.Router("/api/get-terms", &controllers.ApiController{}, "GET:GetTerms")
	web.Router("/api/add-term", &controllers.ApiController{}, "POST:AddTerm")
	web.Router("/api/update-term", &controllers.ApiController{}, "POST:UpdateTerm")
	web.Router("/api/delete-term", &controllers.ApiController{}, "POST:DeleteTerm")

	web.Router("/api/download-translations", &controllers.ApiController{}, "GET:DownloadTranslations")
	web.Router("/api/download-file", &controllers.ApiController{}, "GET:DownloadFile")
	web.Router("/api/upload-source", &controllers.ApiController{}, "POST:UploadSource")
	web.Router("/api/import-translations", &controllers.ApiController{}, "POST:ImportTranslations")
	web.Router("/api/webhook", &controllers.ApiController{}, "POST:Webhook")
}
