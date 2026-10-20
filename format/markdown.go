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

package format

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// MarkdownFormat splits Markdown and MDX files into paragraphs, headings, list items, table rows, block quotes and
// the front matter's title, description and sidebar_label. Code blocks, JSX lines, imports and comments are kept
// as they are, so the translation file is the source file with only the texts replaced.
type MarkdownFormat struct{}

const (
	pieceLiteral     = ""
	pieceParagraph   = "p"
	pieceHeading     = "h"
	pieceListItem    = "li"
	pieceTableRow    = "tr"
	pieceQuote       = "q"
	pieceFrontMatter = "fm"
	pieceAdmonition  = "adm"
)

type mdPiece struct {
	kind string
	// the lines of the source file this piece was made from, as they were (with any "\r")
	original string
	cr       bool
	// for texts: the text, and what goes around each of its lines
	text       string
	prefixes   []string
	restPrefix string
	suffix     string
	// front matter values: the quote character they were written with
	quote string
	key   string
}

var (
	fencePattern     = regexp.MustCompile("^(\\s*)(`{3,}|~{3,})")
	headingPattern   = regexp.MustCompile(`^(\s{0,3}#{1,6}\s+)(.*?)(\s*\{#[^}]*\})?\s*$`)
	listItemPattern  = regexp.MustCompile(`^(\s*)([-*+]|\d{1,9}[.)])(\s+)(\S.*)$`)
	quotePattern     = regexp.MustCompile(`^(\s*>\s?)(.*)$`)
	tableSepPattern  = regexp.MustCompile(`^\s*\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)*\|?\s*$`)
	jsxLinePattern   = regexp.MustCompile(`^\s*(<[A-Za-z/!]|\{/\*|\*/\}|\}\s*$|\)\s*\}?\s*$)`)
	importPattern    = regexp.MustCompile(`^(import|export)\s`)
	imageOnlyPattern = regexp.MustCompile(`^\s*!\[[^\]]*\]\([^)]*\)\s*$`)
	linkDefPattern   = regexp.MustCompile(`^\s*\[[^\]]+\]:\s*\S`)
	hrPattern        = regexp.MustCompile(`^\s*([-*_])(\s*[-*_]){2,}\s*$`)
	admonitionTitle  = regexp.MustCompile(`^(\s*:::[\w-]+\s+)(\S.*?)\s*$`)
	frontMatterKey   = regexp.MustCompile(`^(title|description|sidebar_label)(:\s*)(.*)$`)
	headingIdTail    = regexp.MustCompile(`\s*\{#[^}]*\}\s*$`)
	letterPattern    = regexp.MustCompile(`\pL`)
)

func leadingWhitespace(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

func indentWidth(line string) int {
	width := 0
	for _, r := range leadingWhitespace(line) {
		if r == '\t' {
			width += 4
		} else {
			width++
		}
	}
	return width
}

// startsBlock tells whether a line begins something other than a paragraph continuation.
func startsBlock(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed == "" ||
		fencePattern.MatchString(line) ||
		headingPattern.MatchString(line) ||
		listItemPattern.MatchString(line) ||
		quotePattern.MatchString(line) ||
		strings.HasPrefix(trimmed, "|") ||
		strings.HasPrefix(trimmed, ":::") ||
		strings.HasPrefix(trimmed, "$$") ||
		strings.HasPrefix(trimmed, "<!--") ||
		jsxLinePattern.MatchString(line) ||
		hrPattern.MatchString(line)
}

func hasLetters(s string) bool {
	return letterPattern.MatchString(s)
}

type mdParser struct {
	raw    []string
	lines  []string
	pieces []*mdPiece
}

func (m *mdParser) literal(start int, end int) {
	m.pieces = append(m.pieces, &mdPiece{kind: pieceLiteral, original: strings.Join(m.raw[start:end], "\n")})
}

func (m *mdParser) text(start int, end int, piece *mdPiece) {
	if !hasLetters(piece.text) {
		m.literal(start, end)
		return
	}
	piece.original = strings.Join(m.raw[start:end], "\n")
	piece.cr = strings.HasSuffix(m.raw[start], "\r")
	m.pieces = append(m.pieces, piece)
}

func parseMarkdown(data []byte) []*mdPiece {
	raw := strings.Split(string(data), "\n")
	lines := make([]string, len(raw))
	for i, line := range raw {
		lines[i] = strings.TrimSuffix(line, "\r")
	}
	m := &mdParser{raw: raw, lines: lines}

	i := 0
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		end := -1
		for j := 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "---" {
				end = j
				break
			}
		}
		if end > 0 {
			m.literal(0, 1)
			for j := 1; j < end; j++ {
				piece := parseFrontMatterLine(lines[j])
				if piece == nil {
					m.literal(j, j+1)
				} else {
					m.text(j, j+1, piece)
				}
			}
			m.literal(end, end+1)
			i = end + 1
		}
	}

	inList := false
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		start := i

		switch {
		case trimmed == "":
			i++
			m.literal(start, i)
			continue

		case fencePattern.MatchString(line):
			fence := fencePattern.FindStringSubmatch(line)[2]
			i++
			for i < len(lines) {
				closing := strings.TrimSpace(lines[i])
				i++
				if strings.HasPrefix(closing, fence[:3]) && strings.Trim(closing, fence[:1]) == "" && len(closing) >= len(fence) {
					break
				}
			}
			m.literal(start, i)
			continue

		case strings.HasPrefix(trimmed, "$$"):
			i++
			if !(trimmed != "$$" && strings.HasSuffix(trimmed, "$$")) {
				for i < len(lines) {
					i++
					if strings.HasSuffix(strings.TrimSpace(lines[i-1]), "$$") {
						break
					}
				}
			}
			m.literal(start, i)
			continue

		case strings.HasPrefix(trimmed, "<!--"):
			for i < len(lines) {
				i++
				if strings.Contains(lines[i-1], "-->") {
					break
				}
			}
			m.literal(start, i)
			continue

		case importPattern.MatchString(line):
			// an import or export runs to the next blank line
			for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
				i++
			}
			m.literal(start, i)
			continue

		case admonitionTitle.MatchString(line):
			// ":::tip Don't want to run it yourself?": the title is a text, the rest stays
			match := admonitionTitle.FindStringSubmatch(line)
			i++
			m.text(start, i, &mdPiece{kind: pieceAdmonition, text: match[2], prefixes: []string{match[1]}, restPrefix: match[1]})
			continue

		case jsxLinePattern.MatchString(line) || strings.HasPrefix(trimmed, ":::") || hrPattern.MatchString(line) ||
			imageOnlyPattern.MatchString(line) || linkDefPattern.MatchString(line):
			i++
			m.literal(start, i)
			continue

		case headingPattern.MatchString(line):
			match := headingPattern.FindStringSubmatch(line)
			i++
			m.text(start, i, &mdPiece{kind: pieceHeading, text: match[2], prefixes: []string{match[1]}, restPrefix: match[1], suffix: match[3]})
			inList = false
			continue

		case strings.HasPrefix(trimmed, "|"):
			i++
			if tableSepPattern.MatchString(line) {
				m.literal(start, i)
			} else {
				indent := leadingWhitespace(line)
				m.text(start, i, &mdPiece{kind: pieceTableRow, text: strings.TrimSpace(line), prefixes: []string{indent}, restPrefix: indent})
			}
			continue

		case quotePattern.MatchString(line):
			prefixes := []string{}
			texts := []string{}
			for i < len(lines) && quotePattern.MatchString(lines[i]) {
				match := quotePattern.FindStringSubmatch(lines[i])
				prefixes = append(prefixes, match[1])
				texts = append(texts, match[2])
				i++
			}
			m.text(start, i, &mdPiece{kind: pieceQuote, text: strings.Join(texts, "\n"), prefixes: prefixes, restPrefix: prefixes[len(prefixes)-1]})
			continue

		case listItemPattern.MatchString(line):
			match := listItemPattern.FindStringSubmatch(line)
			first := match[1] + match[2] + match[3]
			prefixes := []string{first}
			texts := []string{match[4]}
			restPrefix := strings.Repeat(" ", len(first))
			i++
			for i < len(lines) && !startsBlock(lines[i]) && indentWidth(lines[i]) >= len(match[1]) {
				indent := leadingWhitespace(lines[i])
				prefixes = append(prefixes, indent)
				texts = append(texts, strings.TrimLeft(lines[i], " \t"))
				restPrefix = indent
				i++
			}
			inList = true
			text := strings.Join(texts, "\n")
			if imageOnlyPattern.MatchString(text) {
				m.literal(start, i)
				continue
			}
			m.text(start, i, &mdPiece{kind: pieceListItem, text: text, prefixes: prefixes, restPrefix: restPrefix})
			continue
		}

		if indentWidth(line) >= 4 && !inList {
			// an indented code block
			i++
			m.literal(start, i)
			continue
		}

		prefixes := []string{}
		texts := []string{}
		for i < len(lines) && (len(texts) == 0 || !startsBlock(lines[i])) {
			prefixes = append(prefixes, leadingWhitespace(lines[i]))
			texts = append(texts, strings.TrimLeft(lines[i], " \t"))
			i++
		}
		if indentWidth(prefixes[0]) == 0 {
			inList = false
		}
		m.text(start, i, &mdPiece{kind: pieceParagraph, text: strings.Join(texts, "\n"), prefixes: prefixes, restPrefix: prefixes[len(prefixes)-1]})
	}
	return m.pieces
}

// parseFrontMatterLine returns the translatable value of a title, description or sidebar_label line, or nil.
func parseFrontMatterLine(line string) *mdPiece {
	match := frontMatterKey.FindStringSubmatch(line)
	if match == nil {
		return nil
	}

	value := strings.TrimRight(match[3], " ")
	quote := ""
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
		quote = value[:1]
		value = value[1 : len(value)-1]
		if quote == "\"" {
			value = strings.ReplaceAll(value, `\"`, `"`)
		} else {
			value = strings.ReplaceAll(value, "''", "'")
		}
	}
	if strings.HasPrefix(value, "|") || strings.HasPrefix(value, ">") {
		return nil
	}
	return &mdPiece{kind: pieceFrontMatter, text: value, key: match[1], prefixes: []string{match[1] + match[2]}, quote: quote}
}

func (p *mdPiece) render(text string) string {
	if text == p.text {
		return p.original
	}

	var res string
	if p.kind == pieceFrontMatter {
		quote := p.quote
		if quote == "" && (strings.ContainsAny(text, ":#\"'{}[]&*!|>%@`") || strings.TrimSpace(text) != text) {
			quote = "\""
		}
		text = strings.ReplaceAll(text, "\n", " ")
		switch quote {
		case "\"":
			text = `"` + strings.ReplaceAll(strings.ReplaceAll(text, `\`, `\\`), `"`, `\"`) + `"`
		case "'":
			text = "'" + strings.ReplaceAll(text, "'", "''") + "'"
		}
		res = p.prefixes[0] + text
	} else {
		if p.kind == pieceHeading || p.kind == pieceTableRow || p.kind == pieceAdmonition {
			text = strings.ReplaceAll(text, "\n", " ")
		}
		lines := strings.Split(text, "\n")
		for i, line := range lines {
			prefix := p.restPrefix
			if i < len(p.prefixes) {
				prefix = p.prefixes[i]
			}
			lines[i] = prefix + line
		}
		res = strings.Join(lines, "\n") + p.suffix
	}

	if p.cr {
		res = strings.ReplaceAll(res, "\n", "\r\n") + "\r"
	}
	return res
}

func pieceKeyBase(p *mdPiece) string {
	sum := sha1.Sum([]byte(p.kind + "\n" + p.text))
	return p.kind + ":" + hex.EncodeToString(sum[:])[:12]
}

// textPieces numbers the texts of a file; a text repeated in the same file gets "#2", "#3", ...
func textPieces(pieces []*mdPiece) ([]*mdPiece, []string) {
	res := []*mdPiece{}
	keys := []string{}
	counts := map[string]int{}
	for _, p := range pieces {
		if p.kind == pieceLiteral {
			continue
		}
		key := pieceKeyBase(p)
		counts[key]++
		if counts[key] > 1 {
			key = fmt.Sprintf("%s#%d", key, counts[key])
		}
		res = append(res, p)
		keys = append(keys, key)
	}
	return res, keys
}

func (f *MarkdownFormat) Parse(source []byte) ([]*Unit, error) {
	texts, keys := textPieces(parseMarkdown(source))

	units := []*Unit{}
	for i, p := range texts {
		note := ""
		switch p.kind {
		case pieceHeading:
			note = "Heading"
		case pieceTableRow:
			note = "Table row: keep the | separators"
		case pieceFrontMatter:
			note = "Front matter: " + p.key
		case pieceAdmonition:
			note = "Title of a note box"
		}
		units = append(units, &Unit{Key: keys[i], Source: p.text, Note: note})
	}
	return units, nil
}

// ParseTranslation lines up the texts of a translated file with the source file's, which works when the
// translation was made from this version of the source: the same blocks in the same order.
func (f *MarkdownFormat) ParseTranslation(source []byte, translation []byte) (map[string]string, error) {
	sourceTexts, keys := textPieces(parseMarkdown(source))
	translationTexts, _ := textPieces(parseMarkdown(translation))

	if len(sourceTexts) != len(translationTexts) {
		return nil, fmt.Errorf("the translation has %d texts but the source has %d", len(translationTexts), len(sourceTexts))
	}
	res := map[string]string{}
	for i, p := range sourceTexts {
		if translationTexts[i].kind != p.kind {
			return nil, fmt.Errorf("text %d is a %s in the source but a %s in the translation", i+1, p.kind, translationTexts[i].kind)
		}
		res[keys[i]] = translationTexts[i].text
	}
	return res, nil
}

func (f *MarkdownFormat) Write(source []byte, translations map[string]string, options *Options) ([]byte, error) {
	pieces := parseMarkdown(source)
	texts, keys := textPieces(pieces)
	keyOf := map[*mdPiece]string{}
	for i, p := range texts {
		keyOf[p] = keys[i]
	}

	translatedCount := 0
	slugs := map[string]int{}
	out := []string{}
	for _, p := range pieces {
		if p.kind == pieceLiteral {
			out = append(out, p.original)
			continue
		}

		text, ok := translations[keyOf[p]]
		if ok {
			translatedCount++
		} else {
			text = p.text
		}

		if p.kind == pieceHeading {
			slug := slugify(p.text)
			slugs[slug]++
			if slugs[slug] > 1 {
				slug = fmt.Sprintf("%s-%d", slug, slugs[slug]-1)
			}
			text = headingIdTail.ReplaceAllString(text, "")
			if options.HeadingIds && p.suffix == "" && slug != "" && text != p.text {
				saved := p.suffix
				p.suffix = " {#" + slug + "}"
				out = append(out, p.render(text))
				p.suffix = saved
				continue
			}
		}
		out = append(out, p.render(text))
	}

	if translatedCount == 0 && !options.FallbackToSource {
		return nil, nil
	}
	return []byte(strings.Join(out, "\n")), nil
}

var (
	slugLinkPattern = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	slugTagPattern  = regexp.MustCompile(`<[^>]+>`)
)

// slugify makes a heading's anchor the way Docusaurus (github-slugger) does.
func slugify(heading string) string {
	s := slugLinkPattern.ReplaceAllString(heading, "$1")
	s = slugTagPattern.ReplaceAllString(s, "")
	s = strings.ToLower(s)

	var b strings.Builder
	for _, r := range s {
		switch {
		case r == ' ':
			b.WriteRune('-')
		case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r):
			b.WriteRune(r)
		}
	}
	return b.String()
}
