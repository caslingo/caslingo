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
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// JsonFormat handles JSON files whose string values are the texts, nested to any depth ({"account": {"Logout": "..."}})
// or flat. With Docusaurus set, the texts are the "message" fields of {"message": ..., "description": ...} objects
// (the files `docusaurus write-translations` makes), and the description becomes the note.
type JsonFormat struct {
	Docusaurus bool
}

type jsonNode struct {
	// 'o' object, 's' string, 'r' anything else (numbers, arrays, ...) kept as the raw JSON
	kind   byte
	keys   []string
	values []*jsonNode
	str    string
	raw    json.RawMessage
}

func parseJson(data []byte) (*jsonNode, error) {
	data = bytes.TrimSpace(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")))
	if len(data) == 0 {
		return &jsonNode{kind: 'o'}, nil
	}

	switch data[0] {
	case '{':
		node := &jsonNode{kind: 'o'}
		decoder := json.NewDecoder(bytes.NewReader(data))
		_, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		for decoder.More() {
			token, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			key, ok := token.(string)
			if !ok {
				return nil, fmt.Errorf("invalid JSON object key: %v", token)
			}
			var value json.RawMessage
			err = decoder.Decode(&value)
			if err != nil {
				return nil, err
			}
			child, err := parseJson(value)
			if err != nil {
				return nil, err
			}
			node.keys = append(node.keys, key)
			node.values = append(node.values, child)
		}
		return node, nil
	case '"':
		node := &jsonNode{kind: 's'}
		err := json.Unmarshal(data, &node.str)
		return node, err
	default:
		if !json.Valid(data) {
			return nil, fmt.Errorf("invalid JSON")
		}
		return &jsonNode{kind: 'r', raw: append(json.RawMessage{}, data...)}, nil
	}
}

func (n *jsonNode) get(key string) *jsonNode {
	for i, k := range n.keys {
		if k == key {
			return n.values[i]
		}
	}
	return nil
}

// isMessage tells whether an object is a Docusaurus {"message": "..."} entry
func (n *jsonNode) isMessage() bool {
	message := n.get("message")
	return n.kind == 'o' && message != nil && message.kind == 's'
}

// EncodeKey turns a path of object keys into a unit key, a JSON array like ["account","Logout"].
func EncodeKey(path []string) string {
	data, _ := json.Marshal(path)
	return string(data)
}

func DecodeKey(key string) []string {
	path := []string{}
	_ = json.Unmarshal([]byte(key), &path)
	return path
}

func (f *JsonFormat) walk(node *jsonNode, path []string, visit func(path []string, node *jsonNode)) {
	if node.kind == 's' {
		visit(path, node)
		return
	}
	if node.kind != 'o' {
		return
	}
	if f.Docusaurus && node.isMessage() {
		visit(path, node)
		return
	}
	for i, key := range node.keys {
		f.walk(node.values[i], append(append([]string{}, path...), key), visit)
	}
}

func (f *JsonFormat) text(node *jsonNode) (string, string) {
	if node.kind == 's' {
		return node.str, ""
	}
	note := ""
	if description := node.get("description"); description != nil && description.kind == 's' {
		note = description.str
	}
	return node.get("message").str, note
}

func (f *JsonFormat) Parse(source []byte) ([]*Unit, error) {
	root, err := parseJson(source)
	if err != nil {
		return nil, err
	}

	units := []*Unit{}
	f.walk(root, nil, func(path []string, node *jsonNode) {
		text, note := f.text(node)
		if f.Docusaurus && node.kind == 's' {
			// a plain string in a Docusaurus file is not a message
			return
		}
		if strings.TrimSpace(text) == "" {
			return
		}
		units = append(units, &Unit{Key: EncodeKey(path), Source: text, Note: note})
	})
	return units, nil
}

func (f *JsonFormat) ParseTranslation(source []byte, translation []byte) (map[string]string, error) {
	root, err := parseJson(translation)
	if err != nil {
		return nil, err
	}

	res := map[string]string{}
	f.walk(root, nil, func(path []string, node *jsonNode) {
		if f.Docusaurus && node.kind == 's' {
			return
		}
		text, _ := f.text(node)
		res[EncodeKey(path)] = text
	})
	return res, nil
}

func (f *JsonFormat) Write(source []byte, translations map[string]string, options *Options) ([]byte, error) {
	root, err := parseJson(source)
	if err != nil {
		return nil, err
	}

	translatedCount := 0
	var build func(node *jsonNode, path []string) *jsonNode
	build = func(node *jsonNode, path []string) *jsonNode {
		key := EncodeKey(path)
		switch {
		case node.kind == 's' && !f.Docusaurus:
			if text, ok := translations[key]; ok {
				translatedCount++
				return &jsonNode{kind: 's', str: text}
			}
			if options.FallbackToSource || strings.TrimSpace(node.str) == "" {
				return node
			}
			return nil
		case node.kind == 'o' && f.Docusaurus && node.isMessage():
			text, ok := translations[key]
			if ok {
				translatedCount++
			} else if !options.FallbackToSource {
				return nil
			} else {
				text = node.get("message").str
			}
			res := &jsonNode{kind: 'o'}
			for i, k := range node.keys {
				value := node.values[i]
				if k == "message" {
					value = &jsonNode{kind: 's', str: text}
				}
				res.keys = append(res.keys, k)
				res.values = append(res.values, value)
			}
			return res
		case node.kind == 'o':
			res := &jsonNode{kind: 'o'}
			for i, k := range node.keys {
				child := build(node.values[i], append(append([]string{}, path...), k))
				if child != nil {
					res.keys = append(res.keys, k)
					res.values = append(res.values, child)
				}
			}
			if len(res.keys) == 0 && len(node.keys) > 0 && !options.FallbackToSource {
				return nil
			}
			return res
		default:
			return node
		}
	}

	res := build(root, nil)
	if translatedCount == 0 && !options.FallbackToSource {
		return nil, nil
	}
	if res == nil {
		res = &jsonNode{kind: 'o'}
	}

	indent := detectIndent(source)
	buf := &bytes.Buffer{}
	writeJson(buf, res, indent, 0)
	if bytes.HasSuffix(bytes.TrimRight(source, " \t"), []byte("\n")) {
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

func detectIndent(data []byte) string {
	for _, line := range strings.Split(string(data), "\n")[1:] {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed != "" {
			indent := line[:len(line)-len(trimmed)]
			if indent != "" {
				return indent
			}
			break
		}
	}
	return "  "
}

func encodeJsonString(s string) string {
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(s)
	return strings.TrimSuffix(buf.String(), "\n")
}

func writeJson(buf *bytes.Buffer, node *jsonNode, indent string, depth int) {
	switch node.kind {
	case 's':
		buf.WriteString(encodeJsonString(node.str))
	case 'r':
		buf.Write(node.raw)
	case 'o':
		if len(node.keys) == 0 {
			buf.WriteString("{}")
			return
		}
		buf.WriteString("{\n")
		for i, key := range node.keys {
			buf.WriteString(strings.Repeat(indent, depth+1))
			buf.WriteString(encodeJsonString(key))
			buf.WriteString(": ")
			writeJson(buf, node.values[i], indent, depth+1)
			if i < len(node.keys)-1 {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		buf.WriteString(strings.Repeat(indent, depth))
		buf.WriteByte('}')
	}
}
