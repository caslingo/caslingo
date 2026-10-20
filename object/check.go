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

import (
	"regexp"
	"slices"
	"strings"
)

const (
	CheckPlaceholders = "placeholders"
	CheckCode         = "code"
	CheckTags         = "tags"
	CheckLinks        = "links"
	CheckWhitespace   = "whitespace"
)

var (
	placeholderPattern = regexp.MustCompile(`%(\d+\$)?[-+#0]*\d*(\.\d+)?[sdvfqxXtTbeEgGcoU]|\{\{\s*[\w.-]+\s*\}\}|\$\{[\w.-]+\}|\{[\w.-]+\}`)
	codePattern        = regexp.MustCompile("`[^`\n]+`")
	tagPattern         = regexp.MustCompile(`</?[A-Za-z][\w.-]*`)
	linkPattern        = regexp.MustCompile(`\]\(([^)\s]+)`)
)

// RunChecks lists what a translation lost or changed compared with its source: placeholders, inline code, tags,
// link targets and the whitespace at both ends. They are what breaks a page when it goes wrong.
func RunChecks(source string, target string) []string {
	res := []string{}
	if target == "" {
		return res
	}

	if !sameMatches(placeholderPattern, source, target) {
		res = append(res, CheckPlaceholders)
	}
	if !sameMatches(codePattern, source, target) {
		res = append(res, CheckCode)
	}
	if !sameMatches(tagPattern, source, target) {
		res = append(res, CheckTags)
	}
	if !sameMatches(linkPattern, source, target) {
		res = append(res, CheckLinks)
	}
	if leadingSpace(source) != leadingSpace(target) || trailingSpace(source) != trailingSpace(target) {
		res = append(res, CheckWhitespace)
	}
	return res
}

func sameMatches(pattern *regexp.Regexp, source string, target string) bool {
	a := pattern.FindAllString(source, -1)
	b := pattern.FindAllString(target, -1)
	if len(a) != len(b) {
		return false
	}
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

func leadingSpace(s string) string {
	return s[:len(s)-len(strings.TrimLeft(s, " \t\n"))]
}

func trailingSpace(s string) string {
	return s[len(strings.TrimRight(s, " \t\n")):]
}
