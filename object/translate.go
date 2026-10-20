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
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/caslingo/caslingo/conf"
	"github.com/xorm-io/builder"
)

// getPendingUnits lists the units of a project that have no translation in the language (or one needing an edit).
func getPendingUnits(project string, language string, options *TranslateOptions) ([]*Unit, error) {
	var translated builder.Cond = builder.Gt{"state": StateEmpty}
	if options.IncludeNeedsEdit {
		translated = builder.Gt{"state": StateNeedsEdit}
	}
	cond := builder.Eq{"project": project}.And(builder.NotIn("id", builder.Select("unit_id").From("translation").
		Where(builder.Eq{"project": project, "language": language}.And(translated))))
	if options.Component != "" {
		cond = cond.And(builder.Eq{"component": options.Component})
	}
	if options.File != "" {
		cond = cond.And(builder.Eq{"file": options.File})
	}

	units := []*Unit{}
	err := ormer.Engine.Where(cond).Asc("component").Asc("file").Asc("position").Find(&units)
	return units, err
}

// getMemoryTexts finds, for each source hash, a translation of the same text elsewhere in the language.
func getMemoryTexts(language string, sourceHashes []string) (map[string]string, error) {
	res := map[string]string{}
	for start := 0; start < len(sourceHashes); start += 500 {
		batch := sourceHashes[start:min(start+500, len(sourceHashes))]
		rows := []struct {
			SourceHash string
			Text       string
		}{}
		args := []interface{}{language, StateTranslated}
		for _, hash := range batch {
			args = append(args, hash)
		}
		query := fmt.Sprintf("SELECT u.source_hash, t.text FROM translation t JOIN unit u ON u.id = t.unit_id "+
			"WHERE t.language = ? AND t.state >= ? AND u.source_hash IN (%s) ORDER BY t.state, t.id", strings.TrimSuffix(strings.Repeat("?,", len(batch)), ","))
		err := ormer.Engine.SQL(query, args...).Find(&rows)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			// approved and newer ones come last and win
			res[row.SourceHash] = row.Text
		}
	}
	return res, nil
}

func runTranslate(ctx context.Context, project *Project, job *Job, languages []string, options *TranslateOptions) error {
	if options == nil {
		options = &TranslateOptions{State: StateTranslated}
	}
	if options.State != StateNeedsEdit {
		options.State = StateTranslated
	}
	if len(languages) == 0 {
		languages = project.Languages
	}
	if !options.MemoryOnly && !IsAiEnabled() {
		return fmt.Errorf("machine translation is not set up on this server (aiBaseUrl, aiModel, aiApiKey)")
	}

	type task struct {
		language string
		units    []*Unit
	}
	tasks := []*task{}
	total := 0
	fromMemory := 0
	for _, language := range languages {
		if !containsString(project.Languages, language) {
			return fmt.Errorf("the project does not have the language %s", language)
		}
		job.setProgress(0, 0, "Looking up the translation memory for "+language)
		units, err := getPendingUnits(project.Name, language, options)
		if err != nil {
			return err
		}

		hashes := []string{}
		for _, unit := range units {
			hashes = append(hashes, unit.SourceHash)
		}
		memory, err := getMemoryTexts(language, hashes)
		if err != nil {
			return err
		}

		rest := []*Unit{}
		for _, unit := range units {
			if text, ok := memory[unit.SourceHash]; ok {
				_, err = SaveTranslation(unit, language, text, StateTranslated, "memory", "")
				if err != nil {
					return err
				}
				fromMemory++
				continue
			}
			rest = append(rest, unit)
		}
		if options.MemoryOnly {
			continue
		}

		batchSize := conf.GetConfigInt("aiBatchSize", 40)
		for start := 0; start < len(rest); {
			// a batch of at most batchSize strings and about 6,000 characters
			end := start
			chars := 0
			for end < len(rest) && end-start < batchSize && (end == start || chars+len(rest[end].Source) < 6000) {
				chars += len(rest[end].Source)
				end++
			}
			tasks = append(tasks, &task{language: language, units: rest[start:end]})
			total += end - start
			start = end
		}
	}

	terms, err := GetTerms(project.Name, "")
	if err != nil {
		return err
	}
	termsByLanguage := map[string][]*Term{}
	for _, term := range terms {
		termsByLanguage[term.Language] = append(termsByLanguage[term.Language], term)
	}

	var mutex sync.Mutex
	done := 0
	translated := 0
	failures := []string{}
	report := func() {
		job.setProgress(done, total, fmt.Sprintf("%d from the translation memory, %d machine translated of %d", fromMemory, translated, total))
	}
	report()

	queue := make(chan *task)
	var wg sync.WaitGroup
	for w := 0; w < max(1, conf.GetConfigInt("aiConcurrency", 4)); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range queue {
				texts, err := MachineTranslate(ctx, project, t.language, t.units, termsByLanguage[t.language])
				count := 0
				if err == nil {
					for _, unit := range t.units {
						text, ok := texts[unit.Id]
						if !ok {
							continue
						}
						state := options.State
						if len(RunChecks(unit.Source, text)) > 0 {
							state = StateNeedsEdit
						}
						_, err = SaveTranslation(unit, t.language, text, state, "machine", "")
						if err != nil {
							break
						}
						count++
					}
				}

				mutex.Lock()
				done += len(t.units)
				translated += count
				if err != nil && ctx.Err() == nil {
					failures = append(failures, err.Error())
				}
				report()
				mutex.Unlock()
			}
		}()
	}

	for _, t := range tasks {
		if ctx.Err() != nil {
			break
		}
		queue <- t
	}
	close(queue)
	wg.Wait()

	if len(failures) > 0 && translated == 0 && total > 0 {
		return fmt.Errorf("machine translation failed: %s", failures[0])
	}
	message := fmt.Sprintf("%d from the translation memory, %d machine translated", fromMemory, translated)
	if total > translated {
		message += fmt.Sprintf(", %d left untranslated", total-translated)
	}
	if len(failures) > 0 {
		message += fmt.Sprintf(" (%d requests failed, e.g. %s)", len(failures), truncate(failures[0], 200))
	}
	job.setProgress(total, total, message)
	return AddChange(&Change{Project: project.Name, Language: strings.Join(languages, ","), Action: ActionMachine, Author: job.Author, NewText: message})
}
