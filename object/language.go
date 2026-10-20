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

import "strings"

// Language is a language code as the files use it ("zh", "zh-CN", "pt-BR", ...) and its names.
type Language struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	NativeName string `json:"nativeName"`
}

var Languages = []*Language{
	{"ar", "Arabic", "العربية"},
	{"az", "Azerbaijani", "Azərbaycan dili"},
	{"bg", "Bulgarian", "Български"},
	{"bn", "Bengali", "বাংলা"},
	{"ca", "Catalan", "Català"},
	{"cs", "Czech", "Čeština"},
	{"da", "Danish", "Dansk"},
	{"de", "German", "Deutsch"},
	{"el", "Greek", "Ελληνικά"},
	{"en", "English", "English"},
	{"es", "Spanish", "Español"},
	{"et", "Estonian", "Eesti"},
	{"fa", "Persian", "فارسی"},
	{"fi", "Finnish", "Suomi"},
	{"fr", "French", "Français"},
	{"he", "Hebrew", "עברית"},
	{"hi", "Hindi", "हिन्दी"},
	{"hr", "Croatian", "Hrvatski"},
	{"hu", "Hungarian", "Magyar"},
	{"id", "Indonesian", "Bahasa Indonesia"},
	{"it", "Italian", "Italiano"},
	{"ja", "Japanese", "日本語"},
	{"kk", "Kazakh", "Қазақ тілі"},
	{"ko", "Korean", "한국어"},
	{"lt", "Lithuanian", "Lietuvių"},
	{"lv", "Latvian", "Latviešu"},
	{"ms", "Malay", "Bahasa Melayu"},
	{"nb", "Norwegian Bokmål", "Norsk bokmål"},
	{"nl", "Dutch", "Nederlands"},
	{"pl", "Polish", "Polski"},
	{"pt", "Portuguese", "Português"},
	{"pt-BR", "Portuguese (Brazil)", "Português (Brasil)"},
	{"ro", "Romanian", "Română"},
	{"ru", "Russian", "Русский"},
	{"sk", "Slovak", "Slovenčina"},
	{"sl", "Slovenian", "Slovenščina"},
	{"sr", "Serbian", "Српски"},
	{"sv", "Swedish", "Svenska"},
	{"th", "Thai", "ไทย"},
	{"tr", "Turkish", "Türkçe"},
	{"uk", "Ukrainian", "Українська"},
	{"ur", "Urdu", "اردو"},
	{"uz", "Uzbek", "Oʻzbekcha"},
	{"vi", "Vietnamese", "Tiếng Việt"},
	{"zh", "Chinese (Simplified)", "简体中文"},
	{"zh-CN", "Chinese (Simplified)", "简体中文"},
	{"zh-TW", "Chinese (Traditional)", "繁體中文"},
	{"zh-HK", "Chinese (Hong Kong)", "繁體中文（香港）"},
}

// GetLanguageName names a code in English for the model, falling back to the base language ("es-ES" -> Spanish).
func GetLanguageName(code string) string {
	for _, language := range Languages {
		if strings.EqualFold(language.Code, code) {
			return language.Name
		}
	}
	base, _, _ := strings.Cut(code, "-")
	base, _, _ = strings.Cut(base, "_")
	for _, language := range Languages {
		if strings.EqualFold(language.Code, base) {
			return language.Name
		}
	}
	return code
}
