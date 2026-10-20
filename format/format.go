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

// Package format reads the translatable strings out of source files and writes translation files back,
// keeping everything else of the source file (order, indentation, code blocks) as it is.
package format

import "fmt"

// Unit is one translatable string of a file. Key identifies it within the file across versions.
type Unit struct {
	Key    string
	Source string
	Note   string
}

type Options struct {
	// untranslated strings are written in the source language instead of being left out
	FallbackToSource bool
	// Markdown: add the source heading's anchor ({#id}) to translated headings that have none
	HeadingIds bool
}

type Format interface {
	// Parse lists the translatable strings of a source file, in file order.
	Parse(source []byte) ([]*Unit, error)
	// ParseTranslation reads an existing translation file of the source file, mapping keys to texts.
	ParseTranslation(source []byte, translation []byte) (map[string]string, error)
	// Write builds the translation file from the source file and the translated texts by key.
	// It returns nil when there is nothing to write (no translated string and no fallback).
	Write(source []byte, translations map[string]string, options *Options) ([]byte, error)
}

type Info struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Extensions  string `json:"extensions"`
}

var Formats = []*Info{
	{Name: "json", DisplayName: "JSON (nested or flat, like i18next)", Extensions: ".json"},
	{Name: "docusaurus-json", DisplayName: "Docusaurus JSON ({\"message\", \"description\"})", Extensions: ".json"},
	{Name: "markdown", DisplayName: "Markdown / MDX", Extensions: ".md, .mdx"},
}

func IsSupported(name string) bool {
	for _, info := range Formats {
		if info.Name == name {
			return true
		}
	}
	return false
}

func Get(name string) (Format, error) {
	switch name {
	case "json":
		return &JsonFormat{}, nil
	case "docusaurus-json":
		return &JsonFormat{Docusaurus: true}, nil
	case "markdown":
		return &MarkdownFormat{}, nil
	}
	return nil, fmt.Errorf("unknown format: %s", name)
}
