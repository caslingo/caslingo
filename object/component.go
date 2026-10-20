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
	"fmt"
	"path"
	"strings"

	"github.com/caslingo/caslingo/format"
)

// Component is one set of files in the repository that share a format, like the UI strings or the docs.
type Component struct {
	Project     string `xorm:"varchar(100) notnull pk" json:"project"`
	Name        string `xorm:"varchar(100) notnull pk" json:"name"`
	DisplayName string `xorm:"varchar(200)" json:"displayName"`
	Position    int    `json:"position"`
	CreatedTime string `xorm:"varchar(100)" json:"createdTime"`

	Format string `xorm:"varchar(50)" json:"format"`
	// SourcePattern matches the source files, like "web/src/locales/en/data.json" or "docs/**/*.md";
	// TranslationPattern is where each language's file goes, like "web/src/locales/{lang}/data.json"
	// or "i18n/{lang}/docusaurus-plugin-content-docs/current/{path}", where {path} is the source file's path below
	// the part of SourcePattern before the first wildcard.
	SourcePattern      string `xorm:"varchar(500)" json:"sourcePattern"`
	TranslationPattern string `xorm:"varchar(500)" json:"translationPattern"`
	ExcludePattern     string `xorm:"varchar(1000)" json:"excludePattern"`
	// the source files come from upload-source (for files generated at build time) instead of the repository
	UploadedSource bool `json:"uploadedSource"`
	// untranslated strings are written in the source language (true), or left out (false)
	FallbackToSource bool `json:"fallbackToSource"`
	// Markdown: give translated headings the source heading's anchor, so links to #anchors keep working
	HeadingIds bool `json:"headingIds"`
}

func GetComponents(project string) ([]*Component, error) {
	components := []*Component{}
	err := ormer.Engine.Where("project = ?", project).Asc("position").Asc("name").Find(&components)
	return components, err
}

func GetComponent(project string, name string) (*Component, error) {
	component := &Component{Project: project, Name: name}
	existed, err := ormer.Engine.Get(component)
	if err != nil || !existed {
		return nil, err
	}
	return component, nil
}

func normalizeComponent(c *Component) error {
	c.Name = strings.TrimSpace(c.Name)
	if !IsValidName(c.Name) {
		return fmt.Errorf("the component name may only have lowercase letters, digits, - and _: %s", c.Name)
	}
	if c.DisplayName == "" {
		c.DisplayName = c.Name
	}
	if !format.IsSupported(c.Format) {
		return fmt.Errorf("unknown format: %s", c.Format)
	}

	c.SourcePattern = strings.Trim(strings.TrimSpace(c.SourcePattern), "/")
	c.TranslationPattern = strings.Trim(strings.TrimSpace(c.TranslationPattern), "/")
	if c.SourcePattern == "" || c.TranslationPattern == "" {
		return fmt.Errorf("the source and translation file patterns are required")
	}
	if !strings.Contains(c.TranslationPattern, "{lang}") {
		return fmt.Errorf("the translation file pattern must contain {lang}")
	}
	if hasWildcard(c.SourcePattern) && !strings.Contains(c.TranslationPattern, "{path}") {
		return fmt.Errorf("the source pattern has wildcards, so the translation file pattern must contain {path}")
	}
	for _, p := range []string{c.SourcePattern, c.TranslationPattern} {
		if strings.Contains(p, "..") || path.IsAbs(p) {
			return fmt.Errorf("file patterns must stay inside the repository: %s", p)
		}
	}
	return nil
}

func AddComponent(component *Component) error {
	err := normalizeComponent(component)
	if err != nil {
		return err
	}

	existing, err := GetComponent(component.Project, component.Name)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("the component %s already exists", component.Name)
	}

	count, err := ormer.Engine.Where("project = ?", component.Project).Count(new(Component))
	if err != nil {
		return err
	}
	component.Position = int(count)
	component.CreatedTime = getCurrentTime()
	_, err = ormer.Engine.Insert(component)
	return err
}

func UpdateComponent(project string, name string, component *Component) error {
	existing, err := GetComponent(project, name)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("the component %s does not exist", name)
	}

	component.Project = project
	component.Name = name
	err = normalizeComponent(component)
	if err != nil {
		return err
	}
	component.CreatedTime = existing.CreatedTime

	_, err = ormer.Engine.ID([]interface{}{project, name}).AllCols().Update(component)
	return err
}

func DeleteComponent(project string, name string) error {
	session := ormer.Engine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return err
	}

	for _, table := range []interface{}{new(Translation), new(Suggestion), new(Unit)} {
		_, err = session.Where("project = ? and component = ?", project, name).Delete(table)
		if err != nil {
			_ = session.Rollback()
			return err
		}
	}
	_, err = session.ID([]interface{}{project, name}).Delete(new(Component))
	if err != nil {
		_ = session.Rollback()
		return err
	}
	return session.Commit()
}
