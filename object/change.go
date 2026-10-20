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

const (
	ActionTranslate = "translate"
	ActionApprove   = "approve"
	ActionSuggest   = "suggest"
	ActionAccept    = "accept"
	ActionMachine   = "machine"
	ActionMemory    = "memory"
	ActionSync      = "sync"
	ActionPush      = "push"
	ActionImport    = "import"
)

// Change is one entry of a project's history: a translation saved, a suggestion made, a sync or a push.
type Change struct {
	Id          int64  `xorm:"pk autoincr" json:"id"`
	Project     string `xorm:"varchar(100) notnull index" json:"project"`
	Component   string `xorm:"varchar(100)" json:"component"`
	UnitId      int64  `xorm:"index" json:"unitId"`
	Language    string `xorm:"varchar(500)" json:"language"`
	Action      string `xorm:"varchar(20)" json:"action"`
	Author      string `xorm:"varchar(100) index" json:"author"`
	OldText     string `xorm:"mediumtext" json:"oldText"`
	NewText     string `xorm:"mediumtext" json:"newText"`
	CreatedTime string `xorm:"varchar(100)" json:"createdTime"`

	Source string `xorm:"-" json:"source"`
}

func AddChange(change *Change) error {
	change.CreatedTime = getCurrentTime()
	_, err := ormer.Engine.Insert(change)
	return err
}

// GetChanges returns a page of the project's history, newest first, with each unit's source text.
func GetChanges(project string, language string, author string, page int, pageSize int) ([]*Change, int64, error) {
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	if page <= 0 {
		page = 1
	}

	session := func() *Change { return &Change{Project: project, Language: language, Author: author} }
	total, err := ormer.Engine.Count(session())
	if err != nil {
		return nil, 0, err
	}

	changes := []*Change{}
	err = ormer.Engine.Desc("id").Limit(pageSize, (page-1)*pageSize).Find(&changes, session())
	if err != nil {
		return nil, 0, err
	}

	ids := []int64{}
	for _, change := range changes {
		if change.UnitId != 0 {
			ids = append(ids, change.UnitId)
		}
	}
	if len(ids) > 0 {
		units := []*Unit{}
		err = ormer.Engine.In("id", ids).Cols("id", "source").Find(&units)
		if err != nil {
			return nil, 0, err
		}
		sources := map[int64]string{}
		for _, unit := range units {
			sources[unit.Id] = unit.Source
		}
		for _, change := range changes {
			change.Source = sources[change.UnitId]
		}
	}
	return changes, total, nil
}

// Contributor is how much one person has done in a project.
type Contributor struct {
	Author string `json:"author"`
	Count  int64  `json:"count"`
}

func GetContributors(project string) ([]*Contributor, error) {
	res := []*Contributor{}
	err := ormer.Engine.Table(new(Change)).Select("author, count(*) as count").
		Where("project = ? and author <> '' and action in (?, ?, ?, ?)", project, ActionTranslate, ActionApprove, ActionSuggest, ActionAccept).
		GroupBy("author").OrderBy("count desc").Limit(50).Find(&res)
	return res, err
}

// GetUnitChanges returns the history of a string in a language, newest first.
func GetUnitChanges(unitId int64, language string) ([]*Change, error) {
	changes := []*Change{}
	err := ormer.Engine.Where("unit_id = ? and language = ?", unitId, language).Desc("id").Limit(20).Find(&changes)
	return changes, err
}
