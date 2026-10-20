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
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/caslingo/caslingo/conf"
)

func hasWildcard(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[{")
}

// globToRegexp supports *, ** (any number of directories), ? and {a,b}.
func globToRegexp(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	inBraces := false
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch {
		case c == '*' && i+1 < len(pattern) && pattern[i+1] == '*':
			if i+2 < len(pattern) && pattern[i+2] == '/' {
				b.WriteString("(?:.*/)?")
				i += 2
			} else {
				b.WriteString(".*")
				i++
			}
		case c == '*':
			b.WriteString("[^/]*")
		case c == '?':
			b.WriteString("[^/]")
		case c == '{':
			b.WriteString("(?:")
			inBraces = true
		case c == '}' && inBraces:
			b.WriteString(")")
			inBraces = false
		case c == ',' && inBraces:
			b.WriteString("|")
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

// patternBase is the directory part of a pattern before its first wildcard: "docs" for "docs/**/*.md".
func patternBase(pattern string) string {
	if !hasWildcard(pattern) {
		return path.Dir(pattern)
	}
	parts := strings.Split(pattern, "/")
	base := []string{}
	for _, part := range parts {
		if hasWildcard(part) {
			break
		}
		base = append(base, part)
	}
	return strings.Join(base, "/")
}

func splitPatterns(s string) []string {
	res := []string{}
	for _, item := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '\n' }) {
		item = strings.Trim(strings.TrimSpace(item), "/")
		if item != "" {
			res = append(res, item)
		}
	}
	return res
}

func getUploadDir(project string) string {
	return filepath.Join(conf.GetConfigString("filesDir"), "uploads", project)
}

// getSourceRoot is where a component's source files are: the repository, or the uploaded files.
func getSourceRoot(project *Project, component *Component) string {
	if component.UploadedSource {
		return getUploadDir(project.Name)
	}
	return project.GetRepoDir()
}

// listSourceFiles finds the files matching the component's source pattern, as slash paths from the root.
func listSourceFiles(root string, component *Component) ([]string, error) {
	if !hasWildcard(component.SourcePattern) {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(component.SourcePattern))); err != nil {
			return []string{}, nil
		}
		return []string{component.SourcePattern}, nil
	}

	include, err := globToRegexp(component.SourcePattern)
	if err != nil {
		return nil, err
	}
	excludes := []*regexp.Regexp{}
	for _, pattern := range splitPatterns(component.ExcludePattern) {
		exclude, err := globToRegexp(pattern)
		if err != nil {
			return nil, err
		}
		excludes = append(excludes, exclude)
	}

	base := patternBase(component.SourcePattern)
	start := filepath.Join(root, filepath.FromSlash(base))
	if _, err := os.Stat(start); err != nil {
		return []string{}, nil
	}

	files := []string{}
	err = filepath.WalkDir(start, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !include.MatchString(rel) {
			return nil
		}
		for _, exclude := range excludes {
			if exclude.MatchString(rel) {
				return nil
			}
		}
		files = append(files, rel)
		return nil
	})
	sort.Strings(files)
	return files, err
}

// GetTranslationPath is where a source file's translation into a language goes in the repository.
func GetTranslationPath(component *Component, sourceFile string, language string) (string, error) {
	res := strings.ReplaceAll(component.TranslationPattern, "{lang}", language)
	if strings.Contains(res, "{path}") {
		base := patternBase(component.SourcePattern)
		rel := sourceFile
		if base != "" && base != "." {
			rel = strings.TrimPrefix(sourceFile, base+"/")
		}
		res = strings.ReplaceAll(res, "{path}", rel)
	}
	res = path.Clean(res)
	if res == "." || strings.HasPrefix(res, "../") || path.IsAbs(res) {
		return "", fmt.Errorf("the translation path of %s leaves the repository: %s", sourceFile, res)
	}
	if res == sourceFile {
		return "", fmt.Errorf("the translation path of %s in %s is the source file itself", sourceFile, language)
	}
	return res, nil
}

func readRepoFile(root string, rel string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

// SaveUploadedSource stores a source file that is generated at build time (like Docusaurus' i18n/en/code.json).
func SaveUploadedSource(project string, rel string, data []byte) error {
	rel = path.Clean(strings.TrimPrefix(filepath.ToSlash(rel), "/"))
	if rel == "." || strings.HasPrefix(rel, "../") {
		return fmt.Errorf("invalid file path: %s", rel)
	}
	target := filepath.Join(getUploadDir(project), filepath.FromSlash(rel))
	err := os.MkdirAll(filepath.Dir(target), 0o755)
	if err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o644)
}
