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

// Stats counts the strings (and their words) of a project, component or language by state.
type Stats struct {
	Language  string `json:"language"`
	Component string `json:"component"`

	Total           int64 `json:"total"`
	TotalWords      int64 `json:"totalWords"`
	Translated      int64 `json:"translated"`
	TranslatedWords int64 `json:"translatedWords"`
	Approved        int64 `json:"approved"`
	NeedsEdit       int64 `json:"needsEdit"`
	Checks          int64 `json:"checks"`
	Suggestions     int64 `json:"suggestions"`
}

type countRow struct {
	Component string
	Language  string
	State     int
	Count     int64
	Words     int64
	Checks    int64
}

// GetProjectStats returns one Stats per language and component, plus per language over all components
// (Component "") and per component over the source (Language "").
func GetProjectStats(project *Project) ([]*Stats, error) {
	totals := []countRow{}
	err := ormer.Engine.SQL("SELECT component, COUNT(*) AS count, COALESCE(SUM(words), 0) AS words FROM unit WHERE project = ? GROUP BY component", project.Name).Find(&totals)
	if err != nil {
		return nil, err
	}

	counts := []countRow{}
	err = ormer.Engine.SQL("SELECT t.component, t.language, t.state, COUNT(*) AS count, COALESCE(SUM(u.words), 0) AS words, "+
		"SUM(CASE WHEN t.checks <> '' THEN 1 ELSE 0 END) AS checks FROM translation t JOIN unit u ON u.id = t.unit_id "+
		"WHERE t.project = ? AND t.state > 0 GROUP BY t.component, t.language, t.state", project.Name).Find(&counts)
	if err != nil {
		return nil, err
	}

	suggestions := []countRow{}
	err = ormer.Engine.SQL("SELECT component, language, COUNT(DISTINCT unit_id) AS count FROM suggestion WHERE project = ? GROUP BY component, language", project.Name).Find(&suggestions)
	if err != nil {
		return nil, err
	}

	stats := map[string]*Stats{}
	get := func(language string, component string) *Stats {
		key := language + "\n" + component
		s := stats[key]
		if s == nil {
			s = &Stats{Language: language, Component: component}
			stats[key] = s
		}
		return s
	}

	for _, row := range totals {
		for _, language := range append([]string{""}, project.Languages...) {
			for _, s := range []*Stats{get(language, row.Component), get(language, "")} {
				s.Total += row.Count
				s.TotalWords += row.Words
			}
		}
	}
	for _, row := range counts {
		if !containsString(project.Languages, row.Language) {
			continue
		}
		for _, s := range []*Stats{get(row.Language, row.Component), get(row.Language, "")} {
			switch {
			case row.State >= StateTranslated:
				s.Translated += row.Count
				s.TranslatedWords += row.Words
				if row.State == StateApproved {
					s.Approved += row.Count
				}
			case row.State == StateNeedsEdit:
				s.NeedsEdit += row.Count
			}
			s.Checks += row.Checks
		}
	}
	for _, row := range suggestions {
		if !containsString(project.Languages, row.Language) {
			continue
		}
		for _, s := range []*Stats{get(row.Language, row.Component), get(row.Language, "")} {
			s.Suggestions += row.Count
		}
	}

	res := []*Stats{}
	for _, language := range append([]string{""}, project.Languages...) {
		res = append(res, get(language, ""))
	}
	components, err := GetComponents(project.Name)
	if err != nil {
		return nil, err
	}
	for _, component := range components {
		for _, language := range append([]string{""}, project.Languages...) {
			res = append(res, get(language, component.Name))
		}
	}
	return res, nil
}
