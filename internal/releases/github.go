// Copyright 2026 The Hugoreleaser Authors
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

package releases

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gohugoio/hugoreleaser/internal/releases/releasetypes"
	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
)

const tokenEnvVar = "GITHUB_TOKEN"

// Validate validates the release type.
func Validate(typ releasetypes.Type) error {
	if typ != releasetypes.GitHub {
		return fmt.Errorf("release: only github is supported for now")
	}
	token := os.Getenv(tokenEnvVar)
	if token == "" {
		return fmt.Errorf("release: missing %q env var", tokenEnvVar)
	}
	return nil
}

func NewClient(ctx context.Context, typ releasetypes.Type) (Client, error) {
	if err := Validate(typ); err != nil {
		return nil, err
	}

	token := os.Getenv(tokenEnvVar)

	// Set in tests to test the all command.
	// and when running with the -try flag.
	if token == "faketoken" {
		return &FakeClient{}, nil
	}

	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	httpClient := oauth2.NewClient(ctx, tokenSource)

	return &GitHubClient{
		client:        github.NewClient(httpClient),
		usernameCache: make(map[string]string),
	}, nil
}

// UploadAssetsFileWithRetries is a wrapper around UploadAssetsFile that retries on temporary errors.
func UploadAssetsFileWithRetries(ctx context.Context, client Client, info ReleaseInfo, releaseID int64, openFile func() (*os.File, error)) error {
	return withRetries(func() (error, bool) {
		f, err := openFile()
		if err != nil {
			return err, false
		}
		defer f.Close()
		err = client.UploadAssetsFile(ctx, info, f, releaseID)
		if err != nil && errors.Is(err, TemporaryError{}) {
			return err, true
		}
		return err, false
	})
}

// UsernameResolver is an interface that allows to resolve the username of a commit.
type UsernameResolver interface {
	ResolveUsername(ctx context.Context, sha, author string, info ReleaseInfo) (string, error)
}

var _ UsernameResolver = &GitHubClient{}

type GitHubClient struct {
	client *github.Client

	usernameCacheMu sync.Mutex
	usernameCache   map[string]string
}

func (c *GitHubClient) ResolveUsername(ctx context.Context, sha, author string, info ReleaseInfo) (string, error) {
	c.usernameCacheMu.Lock()
	defer c.usernameCacheMu.Unlock()
	if username, ok := c.usernameCache[author]; ok {
		return username, nil
	}
	r, resp, err := c.client.Repositories.GetCommit(ctx, info.Settings.RepositoryOwner, info.Settings.Repository, sha, nil)
	if err != nil {
		if resp != nil && (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnprocessableEntity) {
			return "", nil
		}
		return "", err
	}
	if resp != nil && resp.StatusCode != http.StatusOK {
		return "", nil
	}

	if r.Author == nil || r.Author.Login == nil {
		return "", nil
	}

	c.usernameCache[author] = *r.Author.Login
	return c.usernameCache[author], nil
}

func (c *GitHubClient) Release(ctx context.Context, info ReleaseInfo) (int64, error) {
	s := func(s string) *string {
		if s == "" {
			return nil
		}
		return github.String(s)
	}

	settings := info.Settings

	var body string
	releaseNotesSettings := settings.ReleaseNotesSettings

	if releaseNotesSettings.Filename != "" {
		b, err := os.ReadFile(releaseNotesSettings.Filename)
		if err != nil {
			return 0, err
		}
		body = string(b)
	}

	// Truncate body.
	if len(body) > 100000 {
		body = body[:100000]
	}

	r := &github.RepositoryRelease{
		TagName:              s(info.Tag),
		TargetCommitish:      s(info.Commitish),
		Name:                 s(settings.Name),
		Body:                 s(body),
		Draft:                github.Bool(settings.Draft),
		Prerelease:           github.Bool(settings.Prerelease),
		GenerateReleaseNotes: github.Bool(releaseNotesSettings.GenerateOnHost),
	}

	rel, resp, err := c.client.Repositories.CreateRelease(ctx, settings.RepositoryOwner, settings.Repository, r)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("github: unexpected status code: %d", resp.StatusCode)
	}

	return *rel.ID, nil
}

func (c *GitHubClient) UploadAssetsFile(ctx context.Context, info ReleaseInfo, f *os.File, releaseID int64) error {
	settings := info.Settings

	_, resp, err := c.client.Repositories.UploadReleaseAsset(
		ctx,
		settings.RepositoryOwner,
		settings.Repository,
		releaseID,
		&github.UploadOptions{
			Name: filepath.Base(f.Name()),
		},
		f,
	)
	if err == nil {
		return nil
	}

	if resp != nil && !isTemporaryHttpStatus(resp.StatusCode) {
		return err
	}

	return TemporaryError{err}
}

type TemporaryError struct {
	error
}

// Ensure GitHubClient implements PublishClient.
var _ PublishClient = &GitHubClient{}

func (c *GitHubClient) GetReleaseByTag(ctx context.Context, owner, repo, tag string) (int64, bool, error) {
	release, resp, err := c.client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return 0, false, fmt.Errorf("release not found for tag %q", tag)
		}
		return 0, false, err
	}

	return release.GetID(), release.GetDraft(), nil
}

func (c *GitHubClient) PublishRelease(ctx context.Context, owner, repo string, releaseID int64) error {
	_, _, err := c.client.Repositories.EditRelease(ctx, owner, repo, releaseID, &github.RepositoryRelease{
		Draft: github.Bool(false),
	})
	return err
}

func (c *GitHubClient) UpdateFileInRepo(ctx context.Context, owner, repo, path, message string, content []byte) (string, error) {
	// First try to get existing file to get its SHA.
	fileContent, _, resp, err := c.client.Repositories.GetContents(ctx, owner, repo, path, nil)
	var sha string
	if err == nil && fileContent != nil {
		sha = fileContent.GetSHA()
	} else if resp != nil && resp.StatusCode != http.StatusNotFound {
		// Return error only if it's not a 404 (file doesn't exist is OK).
		return "", fmt.Errorf("failed to get file %s: %w", path, err)
	}

	opts := &github.RepositoryContentFileOptions{
		Message: github.String(message),
		Content: content,
	}

	if sha != "" {
		// File exists, update it.
		opts.SHA = github.String(sha)
	}

	result, _, err := c.client.Repositories.CreateFile(ctx, owner, repo, path, opts)
	if err != nil {
		return "", fmt.Errorf("failed to create/update file %s: %w", path, err)
	}

	return result.GetSHA(), nil
}

// isTemporaryHttpStatus returns true if the status code is considered temporary, returning
// true if not sure.
func isTemporaryHttpStatus(status int) bool {
	switch status {
	case http.StatusUnprocessableEntity, http.StatusBadRequest:
		return false
	default:
		return true
	}
}
