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
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/caslingo/caslingo/conf"
	"github.com/caslingo/caslingo/format"
)

func getImportPath(project string) string {
	return filepath.Join(conf.GetConfigString("filesDir"), "imports", project+".zip")
}

// SaveImportZip keeps an uploaded zip of translation files until the import job reads it.
func SaveImportZip(project string, data []byte) error {
	_, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("the file is not a zip: %v", err)
	}
	target := getImportPath(project)
	err = os.MkdirAll(filepath.Dir(target), 0o755)
	if err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o644)
}

func readZipFiles(data []byte) (map[string][]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	res := map[string][]byte{}
	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := path.Clean(strings.TrimPrefix(filepath.ToSlash(f.Name), "./"))
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		res[name] = content
	}
	return res, nil
}

// runImport takes the translations of a zip of translation files laid out as in the repository (an export of
// another platform, like Crowdin's): a text there is taken unless it is the source text or an approved translation
// says otherwise. Only the files of strings Caslingo already has are read, so pull the project first.
func runImport(project *Project, job *Job) error {
	data, err := os.ReadFile(getImportPath(project.Name))
	if err != nil {
		return fmt.Errorf("no uploaded zip to import: %v", err)
	}
	files, err := readZipFiles(data)
	if err != nil {
		return err
	}

	components, err := GetComponents(project.Name)
	if err != nil {
		return err
	}
	result := &SyncResult{}
	matched := 0
	for i, component := range components {
		job.setProgress(i, len(components), "Importing "+component.DisplayName)
		f, err := format.Get(component.Format)
		if err != nil {
			return err
		}
		sourceFiles, err := GetFiles(project.Name, component.Name)
		if err != nil {
			return err
		}
		root := getSourceRoot(project, component)

		for _, file := range sourceFiles {
			source, err := readRepoFile(root, file)
			if err != nil {
				return err
			}
			if source == nil {
				continue
			}
			units, err := getUnitsOfFile(project.Name, component.Name, file)
			if err != nil {
				return err
			}

			for _, language := range project.Languages {
				translationPath, err := GetTranslationPath(component, file, language)
				if err != nil {
					return err
				}
				content, ok := files[translationPath]
				if !ok {
					continue
				}
				matched++
				texts, err := f.ParseTranslation(source, content)
				if err != nil {
					result.warn("%s: %v", translationPath, err)
					continue
				}
				existing, err := getTranslationsOfFile(project.Name, component.Name, file, language)
				if err != nil {
					return err
				}
				err = applyImportedTexts(project, component, language, units, texts, existing, job.Author, result)
				if err != nil {
					return err
				}
			}
		}
	}

	_ = os.Remove(getImportPath(project.Name))
	message := fmt.Sprintf("%d of the %d files in the zip match a translation file; %d translations imported", matched, len(files), result.Imported)
	if len(result.Warnings) > 0 {
		sort.Strings(result.Warnings)
		message += fmt.Sprintf("; %d files could not be lined up with their source (e.g. %s)", len(result.Warnings), result.Warnings[0])
	}
	job.setProgress(1, 1, message)
	return AddChange(&Change{Project: project.Name, Action: ActionImport, Author: job.Author, NewText: message})
}

func applyImportedTexts(project *Project, component *Component, language string, units []*Unit, texts map[string]string, existing map[int64]*Translation, author string, result *SyncResult) error {
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
		if !ok || strings.TrimSpace(text) == "" || text == unit.Source {
			continue
		}
		t := existing[unit.Id]
		if t == nil {
			added = append(added, &Translation{
				UnitId:      unit.Id,
				Language:    language,
				Project:     project.Name,
				Component:   component.Name,
				Text:        text,
				State:       StateTranslated,
				Origin:      "import",
				Author:      author,
				Checks:      strings.Join(RunChecks(unit.Source, text), ","),
				UpdatedTime: now,
			})
			continue
		}
		if t.State == StateApproved || t.Text == text {
			continue
		}

		changes = append(changes, &Change{Project: project.Name, Component: component.Name, UnitId: unit.Id, Language: language, Action: ActionImport, Author: author, OldText: t.Text, NewText: text, CreatedTime: now})
		t.Text = text
		t.State = StateTranslated
		t.Origin = "import"
		t.Author = author
		t.Checks = strings.Join(RunChecks(unit.Source, text), ",")
		t.UpdatedTime = now
		// VcsText stays: the repository does not have this text yet
		_, err = session.ID(t.Id).Cols("text", "state", "origin", "author", "checks", "updated_time").Update(t)
		if err != nil {
			_ = session.Rollback()
			return err
		}
		result.Imported++
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
	for start := 0; start < len(changes); start += 200 {
		batch := changes[start:min(start+200, len(changes))]
		_, err = session.Insert(&batch)
		if err != nil {
			_ = session.Rollback()
			return err
		}
	}
	return session.Commit()
}
