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
	"strings"

	"github.com/xorm-io/builder"
)

const (
	StateEmpty      = 0
	StateNeedsEdit  = 10
	StateTranslated = 20
	StateApproved   = 30
)

// Unit is one source string: a JSON value, or a paragraph, heading or list item of a Markdown file.
type Unit struct {
	Id         int64  `xorm:"pk autoincr" json:"id"`
	Project    string `xorm:"varchar(100) notnull index(idx_unit_file) unique(idx_unit_key)" json:"project"`
	Component  string `xorm:"varchar(100) notnull index(idx_unit_file) unique(idx_unit_key)" json:"component"`
	File       string `xorm:"varchar(500) notnull index(idx_unit_file)" json:"file"`
	Key        string `xorm:"'unit_key' mediumtext" json:"key"`
	KeyHash    string `xorm:"varchar(40) notnull unique(idx_unit_key)" json:"-"`
	Position   int    `json:"position"`
	Source     string `xorm:"mediumtext" json:"source"`
	SourceHash string `xorm:"varchar(40) index" json:"-"`
	Note       string `xorm:"mediumtext" json:"note"`
	Words      int    `json:"words"`

	CreatedTime string `xorm:"varchar(100)" json:"createdTime"`
	UpdatedTime string `xorm:"varchar(100)" json:"updatedTime"`
}

// Translation is a unit's text in one language. A unit with no row in a language is untranslated there.
type Translation struct {
	Id        int64  `xorm:"pk autoincr" json:"id"`
	UnitId    int64  `xorm:"notnull unique(idx_translation_unit)" json:"unitId"`
	Language  string `xorm:"varchar(20) notnull unique(idx_translation_unit) index(idx_translation_project)" json:"language"`
	Project   string `xorm:"varchar(100) notnull index(idx_translation_project)" json:"project"`
	Component string `xorm:"varchar(100) notnull" json:"component"`
	Text      string `xorm:"mediumtext" json:"text"`
	State     int    `xorm:"index" json:"state"`
	// what the repository had for this unit at the last sync, to tell edits made there from edits made here
	VcsText string `xorm:"mediumtext" json:"-"`
	// "human", "suggestion", "memory", "machine" or "repository"
	Origin      string `xorm:"varchar(20)" json:"origin"`
	Author      string `xorm:"varchar(100)" json:"author"`
	Checks      string `xorm:"varchar(500)" json:"checks"`
	UpdatedTime string `xorm:"varchar(100)" json:"updatedTime"`
}

// Suggestion is a translation proposed by someone who cannot translate the project, waiting for a translator.
type Suggestion struct {
	Id          int64  `xorm:"pk autoincr" json:"id"`
	UnitId      int64  `xorm:"notnull index" json:"unitId"`
	Language    string `xorm:"varchar(20) notnull" json:"language"`
	Project     string `xorm:"varchar(100) notnull index" json:"project"`
	Component   string `xorm:"varchar(100) notnull" json:"component"`
	Text        string `xorm:"mediumtext" json:"text"`
	Author      string `xorm:"varchar(100)" json:"author"`
	CreatedTime string `xorm:"varchar(100)" json:"createdTime"`
}

func getUnitKeyHash(file string, key string) string {
	return hashText(file + "\n" + key)
}

func GetUnit(id int64) (*Unit, error) {
	unit := &Unit{}
	existed, err := ormer.Engine.ID(id).Get(unit)
	if err != nil || !existed {
		return nil, err
	}
	return unit, nil
}

func getUnitsOfComponent(project string, component string) ([]*Unit, error) {
	units := []*Unit{}
	err := ormer.Engine.Where("project = ? and component = ?", project, component).Asc("file").Asc("position").Find(&units)
	return units, err
}

func GetTranslation(unitId int64, language string) (*Translation, error) {
	translation := &Translation{}
	existed, err := ormer.Engine.Where("unit_id = ? and language = ?", unitId, language).Get(translation)
	if err != nil || !existed {
		return nil, err
	}
	return translation, nil
}

// getTranslationsOfComponent maps unit id to translation for one language (all languages when language is "").
func getTranslationsOfComponent(project string, component string, language string) (map[int64]*Translation, error) {
	translations := []*Translation{}
	session := ormer.Engine.Where("project = ? and component = ?", project, component)
	if language != "" {
		session = session.And("language = ?", language)
	}
	err := session.Find(&translations)
	if err != nil {
		return nil, err
	}

	res := map[int64]*Translation{}
	for _, t := range translations {
		res[t.UnitId] = t
	}
	return res, nil
}

const (
	FilterAll          = "all"
	FilterUntranslated = "untranslated"
	FilterNeedsEdit    = "needs-edit"
	FilterTranslated   = "translated"
	FilterUnapproved   = "unapproved"
	FilterApproved     = "approved"
	FilterChecks       = "checks"
	FilterSuggestions  = "suggestions"
)

type UnitQuery struct {
	Project   string
	Language  string
	Component string
	File      string
	Filter    string
	Search    string
	Page      int
	PageSize  int
}

// EditorUnit is a unit as the editor shows it: the source, this language's translation and what helps translate it.
type EditorUnit struct {
	*Unit
	Translation *Translation  `json:"translation"`
	Suggestions []*Suggestion `json:"suggestions"`
}

func (q *UnitQuery) condition() builder.Cond {
	cond := builder.NewCond().And(builder.Eq{"project": q.Project})
	if q.Component != "" {
		cond = cond.And(builder.Eq{"component": q.Component})
	}
	if q.File != "" {
		cond = cond.And(builder.Eq{"file": q.File})
	}

	translated := func(where builder.Cond) builder.Cond {
		return builder.In("id", builder.Select("unit_id").From("translation").Where(builder.Eq{"project": q.Project, "language": q.Language}.And(where)))
	}
	switch q.Filter {
	case FilterUntranslated:
		cond = cond.And(builder.NotIn("id", builder.Select("unit_id").From("translation").Where(builder.Eq{"project": q.Project, "language": q.Language}.And(builder.Gt{"state": StateEmpty}))))
	case FilterNeedsEdit:
		cond = cond.And(translated(builder.Eq{"state": StateNeedsEdit}))
	case FilterTranslated:
		cond = cond.And(translated(builder.Gte{"state": StateTranslated}))
	case FilterUnapproved:
		cond = cond.And(translated(builder.Eq{"state": StateTranslated}))
	case FilterApproved:
		cond = cond.And(translated(builder.Eq{"state": StateApproved}))
	case FilterChecks:
		cond = cond.And(translated(builder.Neq{"checks": ""}.And(builder.Gt{"state": StateEmpty})))
	case FilterSuggestions:
		cond = cond.And(builder.In("id", builder.Select("unit_id").From("suggestion").Where(builder.Eq{"project": q.Project, "language": q.Language})))
	}

	search := strings.TrimSpace(q.Search)
	if search != "" {
		like := "%" + search + "%"
		cond = cond.And(builder.Like{"source", search}.
			Or(builder.Like{"unit_key", search}).
			Or(translated(builder.Expr("text LIKE ?", like))))
	}
	return cond
}

// GetEditorUnits returns one page of units with their translations, and how many units match in all.
func GetEditorUnits(q *UnitQuery) ([]*EditorUnit, int64, error) {
	if q.PageSize <= 0 || q.PageSize > 200 {
		q.PageSize = 50
	}
	if q.Page <= 0 {
		q.Page = 1
	}

	cond := q.condition()
	total, err := ormer.Engine.Where(cond).Count(new(Unit))
	if err != nil {
		return nil, 0, err
	}

	units := []*Unit{}
	err = ormer.Engine.Where(cond).Asc("component").Asc("file").Asc("position").Limit(q.PageSize, (q.Page-1)*q.PageSize).Find(&units)
	if err != nil {
		return nil, 0, err
	}

	ids := []int64{}
	for _, unit := range units {
		ids = append(ids, unit.Id)
	}
	translations := []*Translation{}
	suggestions := []*Suggestion{}
	if len(ids) > 0 {
		err = ormer.Engine.In("unit_id", ids).And("language = ?", q.Language).Find(&translations)
		if err != nil {
			return nil, 0, err
		}
		err = ormer.Engine.In("unit_id", ids).And("language = ?", q.Language).Asc("id").Find(&suggestions)
		if err != nil {
			return nil, 0, err
		}
	}

	translationMap := map[int64]*Translation{}
	for _, t := range translations {
		translationMap[t.UnitId] = t
	}
	suggestionMap := map[int64][]*Suggestion{}
	for _, s := range suggestions {
		suggestionMap[s.UnitId] = append(suggestionMap[s.UnitId], s)
	}

	res := []*EditorUnit{}
	for _, unit := range units {
		res = append(res, &EditorUnit{Unit: unit, Translation: translationMap[unit.Id], Suggestions: suggestionMap[unit.Id]})
	}
	return res, total, nil
}

// GetFiles lists the source files of a project (or one component), for the editor's file filter.
func GetFiles(project string, component string) ([]string, error) {
	files := []string{}
	session := ormer.Engine.Table(new(Unit)).Where("project = ?", project)
	if component != "" {
		session = session.And("component = ?", component)
	}
	err := session.Distinct("file").Asc("file").Find(&files)
	return files, err
}

// SaveTranslation stores the text a user (or the machine) gives a unit, recording the change.
func SaveTranslation(unit *Unit, language string, text string, state int, origin string, author string) (*Translation, error) {
	if state < StateEmpty || state > StateApproved {
		return nil, fmt.Errorf("invalid state: %d", state)
	}
	if text == "" {
		state = StateEmpty
	}

	translation, err := GetTranslation(unit.Id, language)
	if err != nil {
		return nil, err
	}

	oldText := ""
	if translation == nil {
		translation = &Translation{
			UnitId:    unit.Id,
			Language:  language,
			Project:   unit.Project,
			Component: unit.Component,
		}
	} else {
		oldText = translation.Text
	}
	oldState := translation.State

	translation.Text = text
	translation.State = state
	translation.Origin = origin
	translation.Author = author
	translation.Checks = strings.Join(RunChecks(unit.Source, text), ",")
	translation.UpdatedTime = getCurrentTime()

	if translation.Id == 0 {
		_, err = ormer.Engine.Insert(translation)
	} else {
		_, err = ormer.Engine.ID(translation.Id).AllCols().Update(translation)
	}
	if err != nil {
		return nil, err
	}

	if oldText != text || oldState != state {
		action := ActionTranslate
		if oldText == text && state == StateApproved {
			action = ActionApprove
		} else if origin == "machine" {
			action = ActionMachine
		} else if origin == "memory" {
			action = ActionMemory
		} else if origin == "suggestion" {
			action = ActionAccept
		}
		err = AddChange(&Change{Project: unit.Project, Component: unit.Component, UnitId: unit.Id, Language: language, Action: action, Author: author, OldText: oldText, NewText: text})
		if err != nil {
			return nil, err
		}
	}
	return translation, nil
}

func AddSuggestion(unit *Unit, language string, text string, author string) (*Suggestion, error) {
	suggestion := &Suggestion{
		UnitId:      unit.Id,
		Language:    language,
		Project:     unit.Project,
		Component:   unit.Component,
		Text:        text,
		Author:      author,
		CreatedTime: getCurrentTime(),
	}
	_, err := ormer.Engine.Insert(suggestion)
	if err != nil {
		return nil, err
	}

	err = AddChange(&Change{Project: unit.Project, Component: unit.Component, UnitId: unit.Id, Language: language, Action: ActionSuggest, Author: author, NewText: text})
	return suggestion, err
}

func GetSuggestion(id int64) (*Suggestion, error) {
	suggestion := &Suggestion{}
	existed, err := ormer.Engine.ID(id).Get(suggestion)
	if err != nil || !existed {
		return nil, err
	}
	return suggestion, nil
}

func DeleteSuggestion(id int64) error {
	_, err := ormer.Engine.ID(id).Delete(new(Suggestion))
	return err
}

// MemoryMatch is a translation of the same source text elsewhere, from any project.
type MemoryMatch struct {
	Source  string `json:"source"`
	Text    string `json:"text"`
	Project string `json:"project"`
	Quality int    `json:"quality"`
}

// GetMemoryMatches finds translations of the same text (100%) and, within the project, of similar texts.
func GetMemoryMatches(unit *Unit, language string) ([]*MemoryMatch, error) {
	res := []*MemoryMatch{}
	seen := map[string]bool{}

	exact := []*Translation{}
	err := ormer.Engine.Where("language = ? and state >= ? and unit_id <> ?", language, StateTranslated, unit.Id).
		And(builder.In("unit_id", builder.Select("id").From("unit").Where(builder.Eq{"source_hash": unit.SourceHash}))).
		Desc("state").Limit(5).Find(&exact)
	if err != nil {
		return nil, err
	}
	for _, t := range exact {
		if !seen[t.Text] {
			seen[t.Text] = true
			res = append(res, &MemoryMatch{Source: unit.Source, Text: t.Text, Project: t.Project, Quality: 100})
		}
	}

	// similar strings: other units of the same file that share words with this one
	words := strings.Fields(unit.Source)
	if len(words) < 3 || len(res) >= 3 {
		return res, nil
	}
	candidates := []*Unit{}
	err = ormer.Engine.Where("project = ? and component = ? and id <> ?", unit.Project, unit.Component, unit.Id).
		And(builder.Like{"source", words[0] + " " + words[1]}).Limit(50).Find(&candidates)
	if err != nil {
		return nil, err
	}
	for _, candidate := range candidates {
		quality := int(similarity(unit.Source, candidate.Source) * 100)
		if quality < 60 {
			continue
		}
		t, err := GetTranslation(candidate.Id, language)
		if err != nil {
			return nil, err
		}
		if t != nil && t.State >= StateTranslated && !seen[t.Text] {
			seen[t.Text] = true
			res = append(res, &MemoryMatch{Source: candidate.Source, Text: t.Text, Project: t.Project, Quality: quality})
		}
		if len(res) >= 5 {
			break
		}
	}
	return res, nil
}

// findMemoryTranslation returns the best translation of exactly this source text in the language, or "".
func findMemoryTranslation(sourceHash string, language string, excludeUnit int64) (string, error) {
	t := &Translation{}
	existed, err := ormer.Engine.Where("language = ? and state >= ? and unit_id <> ?", language, StateTranslated, excludeUnit).
		And(builder.In("unit_id", builder.Select("id").From("unit").Where(builder.Eq{"source_hash": sourceHash}))).
		Desc("state").Desc("id").Get(t)
	if err != nil || !existed {
		return "", err
	}
	return t.Text, nil
}

// GetOtherTranslations returns the string's translations in the other languages, which help with ambiguous texts.
func GetOtherTranslations(unitId int64, language string) ([]*Translation, error) {
	translations := []*Translation{}
	err := ormer.Engine.Where("unit_id = ? and language <> ? and state > ?", unitId, language, StateEmpty).Asc("language").Find(&translations)
	return translations, err
}
