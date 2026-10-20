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
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const randomAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

func randomString(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	for i := range buf {
		buf[i] = randomAlphabet[int(buf[i])%len(randomAlphabet)]
	}
	return string(buf)
}

func getCurrentTime() string {
	return time.Now().Format(time.RFC3339)
}

func hashText(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// IsValidName tells whether a project or component name can go in URLs and file paths as is.
func IsValidName(name string) bool {
	return namePattern.MatchString(name)
}

// countWords counts the words of a string: runs of letters and digits, plus every CJK character.
func countWords(s string) int {
	count := 0
	inWord := false
	for _, r := range s {
		switch {
		case unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r):
			count++
			inWord = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if !inWord {
				count++
				inWord = true
			}
		default:
			inWord = false
		}
	}
	return count
}

// similarity is the share of words two strings have in common (0 to 1).
func similarity(a string, b string) float64 {
	wordsA := strings.Fields(strings.ToLower(a))
	wordsB := strings.Fields(strings.ToLower(b))
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0
	}

	counts := map[string]int{}
	for _, w := range wordsA {
		counts[w]++
	}
	common := 0
	for _, w := range wordsB {
		if counts[w] > 0 {
			counts[w]--
			common++
		}
	}
	return 2 * float64(common) / float64(len(wordsA)+len(wordsB))
}

func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
