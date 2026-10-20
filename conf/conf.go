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

package conf

import (
	"os"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	presetConfigItems := []string{"httpport", "appname"}
	for _, key := range presetConfigItems {
		if value, ok := os.LookupEnv(key); ok {
			err := web.AppConfig.Set(key, value)
			if err != nil {
				panic(err)
			}
		}
	}
}

var defaults = map[string]string{
	"driverName": "sqlite",
	"dbName":     "caslingo.db",
	"webDir":     "web/build",
	"filesDir":   "files",
	"reposDir":   "repos",
	"siteName":   "Caslingo",
}

// GetConfigString reads a config item, letting an environment variable of the same name override app.conf.
func GetConfigString(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	res, _ := web.AppConfig.String(key)
	if res == "" {
		res = defaults[key]
	}
	return res
}

// GetConfigDataSourceName returns dataSourceName, like "root:123456@tcp(localhost:3306)/", to which the ormer appends dbName.
func GetConfigDataSourceName() string {
	dataSourceName := GetConfigString("dataSourceName")
	if GetConfigString("driverName") == "mysql" && dataSourceName != "" && !strings.HasSuffix(dataSourceName, "/") {
		dataSourceName += "/"
	}
	return dataSourceName
}

func GetConfigBool(key string) bool {
	return GetConfigString(key) == "true"
}

func GetConfigInt(key string, defaultValue int) int {
	value, err := strconv.Atoi(GetConfigString(key))
	if err != nil {
		return defaultValue
	}
	return value
}

// GetConfigList splits a comma-separated config item, dropping empty entries.
func GetConfigList(key string) []string {
	res := []string{}
	for _, item := range strings.Split(GetConfigString(key), ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			res = append(res, item)
		}
	}
	return res
}
