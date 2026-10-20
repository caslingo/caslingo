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
	"os"
	"path/filepath"
	"sort"

	"github.com/caslingo/caslingo/format"
)

// ExportedFile is one translation file built from the database.
type ExportedFile struct {
	Path     string
	Content  []byte
	Language string
}

// exportComponent builds the translation files of a component (one language, or all when language is "").
// Only translated and approved texts go out; the rest fall back to the source language or are left out.
func exportComponent(project *Project, component *Component, language string, visit func(file *ExportedFile) error) error {
	f, err := format.Get(component.Format)
	if err != nil {
		return err
	}

	root := getSourceRoot(project, component)
	units, err := getUnitsOfComponent(project.Name, component.Name)
	if err != nil {
		return err
	}
	unitsByFile := map[string][]*Unit{}
	files := []string{}
	for _, unit := range units {
		if unitsByFile[unit.File] == nil {
			files = append(files, unit.File)
		}
		unitsByFile[unit.File] = append(unitsByFile[unit.File], unit)
	}
	sort.Strings(files)

	languages := project.Languages
	if language != "" {
		languages = []string{language}
	}

	options := &format.Options{FallbackToSource: component.FallbackToSource, HeadingIds: component.HeadingIds}
	for _, lang := range languages {
		translations, err := getTranslationsOfComponent(project.Name, component.Name, lang)
		if err != nil {
			return err
		}

		for _, file := range files {
			source, err := readRepoFile(root, file)
			if err != nil {
				return err
			}
			if source == nil {
				continue
			}

			texts := map[string]string{}
			for _, unit := range unitsByFile[file] {
				t := translations[unit.Id]
				if t != nil && t.State >= StateTranslated && t.Text != "" {
					texts[unit.Key] = t.Text
				}
			}

			content, err := f.Write(source, texts, options)
			if err != nil {
				return fmt.Errorf("%s: %v", file, err)
			}
			if content == nil {
				continue
			}
			translationPath, err := GetTranslationPath(component, file, lang)
			if err != nil {
				return err
			}
			err = visit(&ExportedFile{Path: translationPath, Content: content, Language: lang})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// writeTranslationsToRepo writes every translation file into the working tree, returning the paths written.
func writeTranslationsToRepo(project *Project) ([]string, error) {
	components, err := GetComponents(project.Name)
	if err != nil {
		return nil, err
	}

	repoDir := project.GetRepoDir()
	paths := []string{}
	for _, component := range components {
		err = exportComponent(project, component, "", func(file *ExportedFile) error {
			target := filepath.Join(repoDir, filepath.FromSlash(file.Path))
			err := os.MkdirAll(filepath.Dir(target), 0o755)
			if err != nil {
				return err
			}
			old, err := os.ReadFile(target)
			if err == nil && bytes.Equal(old, file.Content) {
				return nil
			}
			paths = append(paths, file.Path)
			return os.WriteFile(target, file.Content, 0o644)
		})
		if err != nil {
			return nil, fmt.Errorf("%s: %v", component.Name, err)
		}
	}
	return paths, nil
}

// ExportZip builds a zip of the project's translation files, at their paths in the repository, for CI jobs that
// take translations at build time instead of from the repository.
func ExportZip(project *Project, componentName string, language string) ([]byte, error) {
	components, err := GetComponents(project.Name)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	writer := zip.NewWriter(buf)
	for _, component := range components {
		if componentName != "" && component.Name != componentName {
			continue
		}
		err = exportComponent(project, component, language, func(file *ExportedFile) error {
			w, err := writer.Create(file.Path)
			if err != nil {
				return err
			}
			_, err = w.Write(file.Content)
			return err
		})
		if err != nil {
			return nil, err
		}
	}
	err = writer.Close()
	return buf.Bytes(), err
}

// ExportFile builds one translation file, for downloading from the editor.
func ExportFile(project *Project, component *Component, file string, language string) (*ExportedFile, error) {
	var res *ExportedFile
	err := exportComponent(project, component, language, func(f *ExportedFile) error {
		translationPath, err := GetTranslationPath(component, file, language)
		if err != nil {
			return err
		}
		if f.Path == translationPath {
			res = f
		}
		return nil
	})
	return res, err
}
