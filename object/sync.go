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

	"github.com/caslingo/caslingo/format"
	"github.com/xorm-io/builder"
	"github.com/xorm-io/xorm"
)

// SyncResult tells what a sync changed.
type SyncResult struct {
	Files    int      `json:"files"`
	Added    int      `json:"added"`
	Changed  int      `json:"changed"`
	Removed  int      `json:"removed"`
	Imported int      `json:"imported"`
	Warnings []string `json:"warnings"`
}

func (r *SyncResult) String() string {
	s := fmt.Sprintf("%d files: %d new strings, %d changed, %d removed, %d translations taken from the repository", r.Files, r.Added, r.Changed, r.Removed, r.Imported)
	if len(r.Warnings) > 0 {
		s += fmt.Sprintf(" (%d warnings)", len(r.Warnings))
	}
	return s
}

func (r *SyncResult) warn(format string, args ...interface{}) {
	if len(r.Warnings) < 100 {
		r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
	}
}

// syncComponent reads the component's source files into units, then takes the translations the repository has
// that were made or changed there (see importTranslations).
func syncComponent(project *Project, component *Component, result *SyncResult) error {
	f, err := format.Get(component.Format)
	if err != nil {
		return err
	}

	root := getSourceRoot(project, component)
	files, err := listSourceFiles(root, component)
	if err != nil {
		return err
	}
	result.Files += len(files)

	existingUnits, err := getUnitsOfComponent(project.Name, component.Name)
	if err != nil {
		return err
	}
	existing := map[string]*Unit{}
	for _, unit := range existingUnits {
		existing[unit.KeyHash] = unit
	}

	seen := map[string]bool{}
	for _, file := range files {
		data, err := readRepoFile(root, file)
		if err != nil {
			return err
		}
		parsed, err := f.Parse(data)
		if err != nil {
			result.warn("%s: %v", file, err)
			// keep the file's units as they are rather than deleting them all
			for _, unit := range existingUnits {
				if unit.File == file {
					seen[unit.KeyHash] = true
				}
			}
			continue
		}

		err = syncFile(project, component, file, parsed, existing, seen, result)
		if err != nil {
			return err
		}

		units, err := getUnitsOfFile(project.Name, component.Name, file)
		if err != nil {
			return err
		}
		err = importTranslations(project, component, f, file, data, units, result)
		if err != nil {
			return err
		}
	}

	removed := []int64{}
	for keyHash, unit := range existing {
		if !seen[keyHash] {
			removed = append(removed, unit.Id)
		}
	}
	result.Removed += len(removed)
	return deleteUnits(removed)
}

func getUnitsOfFile(project string, component string, file string) ([]*Unit, error) {
	units := []*Unit{}
	err := ormer.Engine.Where("project = ? and component = ? and file = ?", project, component, file).Asc("position").Find(&units)
	return units, err
}

func deleteUnits(ids []int64) error {
	for start := 0; start < len(ids); start += 500 {
		end := min(start+500, len(ids))
		batch := ids[start:end]
		for _, table := range []interface{}{new(Translation), new(Suggestion)} {
			_, err := ormer.Engine.In("unit_id", batch).Delete(table)
			if err != nil {
				return err
			}
		}
		_, err := ormer.Engine.In("id", batch).Delete(new(Unit))
		if err != nil {
			return err
		}
	}
	return nil
}

func syncFile(project *Project, component *Component, file string, parsed []*format.Unit, existing map[string]*Unit, seen map[string]bool, result *SyncResult) error {
	session := ormer.Engine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return err
	}

	now := getCurrentTime()
	added := []*Unit{}
	for position, p := range parsed {
		keyHash := getUnitKeyHash(file, p.Key)
		if seen[keyHash] {
			result.warn("%s: the key %s appears twice", file, p.Key)
			continue
		}
		seen[keyHash] = true

		unit := existing[keyHash]
		if unit == nil {
			unit = &Unit{
				Project:     project.Name,
				Component:   component.Name,
				File:        file,
				Key:         p.Key,
				KeyHash:     keyHash,
				Position:    position,
				Source:      p.Source,
				SourceHash:  hashText(p.Source),
				Note:        p.Note,
				Words:       countWords(p.Source),
				CreatedTime: now,
				UpdatedTime: now,
			}
			added = append(added, unit)
			continue
		}

		if unit.Source != p.Source {
			unit.Source = p.Source
			unit.SourceHash = hashText(p.Source)
			unit.Words = countWords(p.Source)
			unit.UpdatedTime = now
			result.Changed++
			// the translations are of the old text now: someone has to look at them again
			_, err = session.Where("unit_id = ? and state > ?", unit.Id, StateNeedsEdit).Cols("state").Update(&Translation{State: StateNeedsEdit})
			if err != nil {
				_ = session.Rollback()
				return err
			}
		} else if unit.Position == position && unit.Note == p.Note {
			continue
		}
		unit.Position = position
		unit.Note = p.Note
		_, err = session.ID(unit.Id).Cols("source", "source_hash", "words", "position", "note", "updated_time").Update(unit)
		if err != nil {
			_ = session.Rollback()
			return err
		}
	}

	if len(added) > 0 {
		for start := 0; start < len(added); start += 200 {
			batch := added[start:min(start+200, len(added))]
			_, err = session.Insert(&batch)
			if err != nil {
				_ = session.Rollback()
				return err
			}
		}
		result.Added += len(added)

		err = carryOverTranslations(session, file, added, existing, seen)
		if err != nil {
			_ = session.Rollback()
			return err
		}
	}
	return session.Commit()
}

// carryOverTranslations gives a new unit the translations of a removed unit of the same file whose text was
// almost the same (a Markdown paragraph with one word changed gets a new key), marked as needing an edit.
func carryOverTranslations(session *xorm.Session, file string, added []*Unit, existing map[string]*Unit, seen map[string]bool) error {
	removed := []*Unit{}
	for keyHash, unit := range existing {
		if unit.File == file && !seen[keyHash] {
			removed = append(removed, unit)
		}
	}
	if len(removed) == 0 {
		return nil
	}

	// a batch insert does not give back the ids
	keyHashes := []string{}
	for _, unit := range added {
		keyHashes = append(keyHashes, unit.KeyHash)
	}
	inserted := []*Unit{}
	err := session.Where("project = ? and component = ?", added[0].Project, added[0].Component).In("key_hash", keyHashes).Find(&inserted)
	if err != nil {
		return err
	}

	for _, unit := range inserted {
		var best *Unit
		bestScore := 0.75
		for _, old := range removed {
			score := similarity(unit.Source, old.Source)
			if score > bestScore {
				best = old
				bestScore = score
			}
		}
		if best == nil {
			continue
		}

		translations := []*Translation{}
		err := session.Where("unit_id = ? and state > ?", best.Id, StateEmpty).Find(&translations)
		if err != nil {
			return err
		}
		for _, t := range translations {
			t.Id = 0
			t.UnitId = unit.Id
			t.State = StateNeedsEdit
			t.Checks = strings.Join(RunChecks(unit.Source, t.Text), ",")
		}
		if len(translations) > 0 {
			_, err = session.Insert(&translations)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// importTranslations reads each language's translation file of a source file from the repository. A text there
// is taken when it is new, or when it changed in the repository since the last sync while nobody changed it here;
// when both changed, the one here wins and goes to the repository with the next push.
func importTranslations(project *Project, component *Component, f format.Format, file string, source []byte, units []*Unit, result *SyncResult) error {
	if len(units) == 0 {
		return nil
	}
	repoDir := project.GetRepoDir()

	for _, language := range project.Languages {
		translationPath, err := GetTranslationPath(component, file, language)
		if err != nil {
			result.warn("%v", err)
			continue
		}
		data, err := readRepoFile(repoDir, translationPath)
		if err != nil {
			return err
		}
		if data == nil {
			continue
		}
		texts, err := f.ParseTranslation(source, data)
		if err != nil {
			result.warn("%s: %v", translationPath, err)
			continue
		}

		existing, err := getTranslationsOfFile(project.Name, component.Name, file, language)
		if err != nil {
			return err
		}
		err = applyRepoTexts(project, component, language, units, texts, existing, result)
		if err != nil {
			return err
		}
	}
	return nil
}

func getTranslationsOfFile(project string, component string, file string, language string) (map[int64]*Translation, error) {
	translations := []*Translation{}
	err := ormer.Engine.Where("project = ? and component = ? and language = ?", project, component, language).
		And(builder.In("unit_id", builder.Select("id").From("unit").Where(builder.Eq{"project": project, "component": component, "file": file}))).
		Find(&translations)
	if err != nil {
		return nil, err
	}

	res := map[int64]*Translation{}
	for _, t := range translations {
		res[t.UnitId] = t
	}
	return res, nil
}

func applyRepoTexts(project *Project, component *Component, language string, units []*Unit, texts map[string]string, existing map[int64]*Translation, result *SyncResult) error {
	session := ormer.Engine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return err
	}

	now := getCurrentTime()
	added := []*Translation{}
	changes := []*Change{}
	for _, unit := range units {
		text, ok := texts[unit.Key]
		if !ok {
			continue
		}
		t := existing[unit.Id]

		if t == nil {
			// an untranslated string in a file that falls back to the source language
			if strings.TrimSpace(text) == "" || text == unit.Source {
				continue
			}
			added = append(added, &Translation{
				UnitId:      unit.Id,
				Language:    language,
				Project:     project.Name,
				Component:   component.Name,
				Text:        text,
				State:       StateTranslated,
				VcsText:     text,
				Origin:      "repository",
				Checks:      strings.Join(RunChecks(unit.Source, text), ","),
				UpdatedTime: now,
			})
			continue
		}

		switch {
		case text == t.VcsText:
			continue
		case text == t.Text || text == unit.Source || strings.TrimSpace(text) == "" || t.Text != t.VcsText:
			// already the same here, reset to the source text, or edited here too: only remember what the repository has
			t.VcsText = text
			_, err = session.ID(t.Id).Cols("vcs_text").Update(t)
		default:
			changes = append(changes, &Change{Project: project.Name, Component: component.Name, UnitId: unit.Id, Language: language, Action: ActionSync, OldText: t.Text, NewText: text, CreatedTime: now})
			t.Text = text
			t.VcsText = text
			t.State = StateTranslated
			t.Origin = "repository"
			t.Author = ""
			t.Checks = strings.Join(RunChecks(unit.Source, text), ",")
			t.UpdatedTime = now
			_, err = session.ID(t.Id).AllCols().Update(t)
			result.Imported++
		}
		if err != nil {
			_ = session.Rollback()
			return err
		}
	}

	for start := 0; start < len(added); start += 200 {
		batch := added[start:min(start+200, len(added))]
		_, err = session.Insert(&batch)
		if err != nil {
			_ = session.Rollback()
			return err
		}
	}
	result.Imported += len(added)
	if len(changes) > 0 {
		_, err = session.Insert(&changes)
		if err != nil {
			_ = session.Rollback()
			return err
		}
	}
	return session.Commit()
}
