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
)

// Term is a glossary entry: how a word of the product is translated in one language, or that it stays as is.
type Term struct {
	Id          int64  `xorm:"pk autoincr" json:"id"`
	Project     string `xorm:"varchar(100) notnull index" json:"project"`
	Source      string `xorm:"varchar(200) notnull" json:"source"`
	Language    string `xorm:"varchar(20) notnull" json:"language"`
	Target      string `xorm:"varchar(200)" json:"target"`
	Note        string `xorm:"varchar(500)" json:"note"`
	CreatedTime string `xorm:"varchar(100)" json:"createdTime"`
}

func GetTerms(project string, language string) ([]*Term, error) {
	terms := []*Term{}
	session := ormer.Engine.Where("project = ?", project)
	if language != "" {
		session = session.And("language = ?", language)
	}
	err := session.Asc("source").Find(&terms)
	return terms, err
}

func GetTerm(id int64) (*Term, error) {
	term := &Term{}
	existed, err := ormer.Engine.ID(id).Get(term)
	if err != nil || !existed {
		return nil, err
	}
	return term, nil
}

func AddTerm(term *Term) error {
	term.Source = strings.TrimSpace(term.Source)
	term.Target = strings.TrimSpace(term.Target)
	if term.Source == "" || term.Language == "" {
		return fmt.Errorf("the term and the language are required")
	}
	term.CreatedTime = getCurrentTime()
	_, err := ormer.Engine.Insert(term)
	return err
}

func UpdateTerm(id int64, term *Term) error {
	term.Source = strings.TrimSpace(term.Source)
	term.Target = strings.TrimSpace(term.Target)
	_, err := ormer.Engine.ID(id).Cols("source", "language", "target", "note").Update(term)
	return err
}

func DeleteTerm(id int64) error {
	_, err := ormer.Engine.ID(id).Delete(new(Term))
	return err
}

// matchTerms picks the glossary entries that appear in a text, matching whole words without case.
func matchTerms(terms []*Term, text string) []*Term {
	res := []*Term{}
	lower := strings.ToLower(text)
	for _, term := range terms {
		source := strings.ToLower(term.Source)
		index := strings.Index(lower, source)
		for index >= 0 {
			before := index == 0 || !isWordChar(lower[index-1])
			after := index+len(source) >= len(lower) || !isWordChar(lower[index+len(source)])
			if before && after {
				res = append(res, term)
				break
			}
			next := strings.Index(lower[index+1:], source)
			if next < 0 {
				break
			}
			index += next + 1
		}
	}
	return res
}

func isWordChar(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_'
}

// GetMatchingTerms returns the glossary entries of the language that appear in a text.
func GetMatchingTerms(project string, language string, text string) ([]*Term, error) {
	terms, err := GetTerms(project, language)
	if err != nil {
		return nil, err
	}
	return matchTerms(terms, text), nil
}
