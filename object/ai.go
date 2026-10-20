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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/caslingo/caslingo/conf"
)

var aiHttpClient = &http.Client{Timeout: 5 * time.Minute}

func IsAiEnabled() bool {
	return conf.GetConfigString("aiBaseUrl") != "" && conf.GetConfigString("aiModel") != "" && conf.GetConfigString("aiApiKey") != ""
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// callChat sends one chat completion request to the OpenAI-compatible API in app.conf and returns the answer.
func callChat(ctx context.Context, messages []chatMessage, jsonOutput bool) (string, error) {
	if !IsAiEnabled() {
		return "", fmt.Errorf("machine translation is not set up: aiBaseUrl, aiModel and aiApiKey are empty")
	}

	body := map[string]interface{}{
		"model":    conf.GetConfigString("aiModel"),
		"messages": messages,
	}
	if jsonOutput {
		body["response_format"] = map[string]string{"type": "json_object"}
	}
	// provider-specific fields, e.g. {"enable_thinking": false} for Qwen3 on DashScope
	if extra := conf.GetConfigString("aiExtraBody"); extra != "" {
		extraFields := map[string]interface{}{}
		err := json.Unmarshal([]byte(extra), &extraFields)
		if err != nil {
			return "", fmt.Errorf("aiExtraBody is not a JSON object: %v", err)
		}
		for k, v := range extraFields {
			body[k] = v
		}
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	url := strings.TrimSuffix(conf.GetConfigString("aiBaseUrl"), "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+conf.GetConfigString("aiApiKey"))
	resp, err := aiHttpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	err = json.Unmarshal(respData, &res)
	if err != nil {
		return "", fmt.Errorf("the model API returned %d: %s", resp.StatusCode, truncate(string(respData), 300))
	}
	if res.Error != nil {
		return "", fmt.Errorf("the model API: %s", res.Error.Message)
	}
	if len(res.Choices) == 0 {
		return "", fmt.Errorf("the model API returned no answer (%d)", resp.StatusCode)
	}
	return res.Choices[0].Message.Content, nil
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

type aiItem struct {
	Id      int    `json:"id"`
	Text    string `json:"text"`
	Context string `json:"context,omitempty"`
}

// getTranslatePrompt tells the model how to translate the project's strings.
func getTranslatePrompt(project *Project, language string, terms []*Term) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You translate the user interface and documentation of the software product \"%s\" from %s to %s.\n",
		project.DisplayName, GetLanguageName(project.SourceLanguage), GetLanguageName(language))
	if project.Description != "" {
		fmt.Fprintf(&b, "About the product: %s\n", project.Description)
	}
	b.WriteString(`
The user sends JSON {"items": [{"id": 1, "text": "...", "context": "..."}]}. Answer only with JSON
{"items": [{"id": 1, "text": "<translation>"}]}, one item for every item sent, with the same ids.

Rules:
- Translate the meaning naturally, as a native technical writer would, keeping the tone of the original.
- Keep unchanged: placeholders such as {name}, {{count}}, %s, %d, %v, ${var}; inline code in backticks; HTML and JSX tags
  with their attributes; URLs and link targets; Markdown syntax (**, _, #, -, |, >, [text](url) where only the text is
  translated); admonition keywords like :::tip; heading anchors like {#id}; emoji.
- Keep the same leading and trailing spaces and the same line breaks as the original.
- Do not translate code, command lines, file paths, configuration keys, API names, or product and brand names.
- "context" tells where the text is (a heading, a table row, the key of a UI string); do not translate it.
- If a text needs no translation (a name, a code identifier, a number), return it unchanged.
`)
	if len(terms) > 0 {
		b.WriteString("\nGlossary (use these translations; an empty translation means keep the term as is):\n")
		for _, term := range terms {
			target := term.Target
			if target == "" {
				target = term.Source
			}
			fmt.Fprintf(&b, "- %s => %s", term.Source, target)
			if term.Note != "" {
				fmt.Fprintf(&b, " (%s)", term.Note)
			}
			b.WriteString("\n")
		}
	}
	if strings.TrimSpace(project.Instructions) != "" {
		b.WriteString("\nThe project's own instructions:\n")
		b.WriteString(strings.TrimSpace(project.Instructions))
		b.WriteString("\n")
	}
	return b.String()
}

func getUnitContext(unit *Unit) string {
	context := unit.Note
	if strings.HasPrefix(unit.Key, "[") {
		key := strings.Join(decodeKey(unit.Key), " > ")
		if context == "" {
			context = "key: " + key
		} else {
			context = "key: " + key + "; " + context
		}
	}
	return context
}

func decodeKey(key string) []string {
	path := []string{}
	_ = json.Unmarshal([]byte(key), &path)
	return path
}

// MachineTranslate translates units into a language in one request, returning the texts by unit id.
// Texts the model left out are missing from the result.
func MachineTranslate(ctx context.Context, project *Project, language string, units []*Unit, terms []*Term) (map[int64]string, error) {
	items := []aiItem{}
	all := ""
	for i, unit := range units {
		items = append(items, aiItem{Id: i + 1, Text: unit.Source, Context: getUnitContext(unit)})
		all += unit.Source + "\n"
	}
	input, err := json.Marshal(map[string]interface{}{"items": items})
	if err != nil {
		return nil, err
	}

	messages := []chatMessage{
		{Role: "system", Content: getTranslatePrompt(project, language, matchTerms(terms, all))},
		{Role: "user", Content: string(input)},
	}

	var answer string
	for attempt := 0; attempt < 3; attempt++ {
		answer, err = callChat(ctx, messages, true)
		if err == nil || ctx.Err() != nil {
			break
		}
		time.Sleep(time.Duration(attempt+1) * 3 * time.Second)
	}
	if err != nil {
		return nil, err
	}

	answer = strings.TrimSpace(answer)
	answer = strings.TrimPrefix(strings.TrimSuffix(strings.TrimPrefix(answer, "```json"), "```"), "```")
	var output struct {
		Items []aiItem `json:"items"`
	}
	err = json.Unmarshal([]byte(strings.TrimSpace(answer)), &output)
	if err != nil {
		return nil, fmt.Errorf("the model did not answer with JSON: %s", truncate(answer, 200))
	}

	res := map[int64]string{}
	for _, item := range output.Items {
		if item.Id < 1 || item.Id > len(units) || item.Text == "" {
			continue
		}
		unit := units[item.Id-1]
		// models drop the spaces around a string; they matter when strings are joined
		res[unit.Id] = leadingSpace(unit.Source) + strings.TrimSpace(item.Text) + trailingSpace(unit.Source)
	}
	return res, nil
}
