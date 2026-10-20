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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var githubHttpClient = &http.Client{Timeout: 30 * time.Second}

// getGithubRepo returns "owner/repo" for a github.com URL, or "".
func getGithubRepo(repoUrl string) string {
	u, err := url.Parse(repoUrl)
	if err != nil || !strings.EqualFold(u.Host, "github.com") {
		return ""
	}
	parts := strings.Split(strings.Trim(strings.TrimSuffix(u.Path, ".git"), "/"), "/")
	if len(parts) != 2 {
		return ""
	}
	return parts[0] + "/" + parts[1]
}

func githubRequest(project *Project, method string, path string, body interface{}, res interface{}) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, "https://api.github.com"+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+project.RepoToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := githubHttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub %s %s returned %d: %s", method, path, resp.StatusCode, string(data))
	}
	if res != nil {
		return json.Unmarshal(data, res)
	}
	return nil
}

// ensurePullRequest opens a pull request from the push branch, unless one is already open (a force push updated it).
func ensurePullRequest(project *Project) (string, error) {
	repo := getGithubRepo(project.RepoUrl)
	if repo == "" {
		return "", fmt.Errorf("pull requests are only supported for github.com repositories; the branch %s was pushed", project.PushBranch)
	}
	owner, _, _ := strings.Cut(repo, "/")

	var pulls []struct {
		HtmlUrl string `json:"html_url"`
	}
	query := url.Values{"head": {owner + ":" + project.PushBranch}, "base": {project.Branch}, "state": {"open"}}
	err := githubRequest(project, http.MethodGet, "/repos/"+repo+"/pulls?"+query.Encode(), nil, &pulls)
	if err != nil {
		return "", err
	}
	if len(pulls) > 0 {
		return pulls[0].HtmlUrl, nil
	}

	var pull struct {
		HtmlUrl string `json:"html_url"`
	}
	body := map[string]string{
		"title": project.CommitMessage,
		"head":  project.PushBranch,
		"base":  project.Branch,
		"body":  "Translations from Caslingo.",
	}
	err = githubRequest(project, http.MethodPost, "/repos/"+repo+"/pulls", body, &pull)
	return pull.HtmlUrl, err
}

// VerifyGithubSignature checks the X-Hub-Signature-256 header of a webhook against the project's secret.
func VerifyGithubSignature(secret string, body []byte, signature string) bool {
	if secret == "" || !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(strings.TrimPrefix(signature, "sha256=")))
}
