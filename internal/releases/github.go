package releases

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bep/hugoreleaser/internal/config"
	"github.com/bep/hugoreleaser/internal/releases/releasetypes"
	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
)

const tokenEnvVar = "GITHUB_TOKEN"

func NewClient(ctx context.Context, typ releasetypes.Type) (Client, error) {
	if typ != releasetypes.GitHub {
		return nil, fmt.Errorf("github: only github is supported for now")
	}
	token := os.Getenv(tokenEnvVar)
	if token == "" {
		return nil, fmt.Errorf("github: missing %q env var", tokenEnvVar)
	}

	// Set in tests to test the all command.
	// We cannot curently use the -try flag because
	// that does not create any archives.
	if token == "faketoken" {
		return &FakeClient{}, nil
	}

	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	httpClient := oauth2.NewClient(ctx, tokenSource)

	return &GitHubClient{
		client: github.NewClient(httpClient),
	}, nil

}

type Client interface {
	Release(ctx context.Context, tagName, committish string, settings config.ReleaseSettings) (int64, error)
	UploadAssetsFile(ctx context.Context, settings config.ReleaseSettings, f *os.File, releaseID int64) error
}

type GitHubClient struct {
	client *github.Client
}

func (c GitHubClient) Release(ctx context.Context, tagName, committish string, settings config.ReleaseSettings) (int64, error) {
	s := func(s string) *string {
		if s == "" {
			return nil
		}
		return github.String(s)
	}

	var body string

	if settings.ReleaseNotesFilename != "" {
		b, err := os.ReadFile(settings.ReleaseNotesFilename)
		if err != nil {
			return 0, err
		}
		body = string(b)
	}

	r := &github.RepositoryRelease{
		TagName:              s(tagName),
		TargetCommitish:      s(committish),
		Name:                 s(settings.Name),
		Body:                 s(body),
		Draft:                github.Bool(settings.Draft),
		Prerelease:           github.Bool(settings.Prerelease),
		GenerateReleaseNotes: github.Bool(settings.GenerateReleaseNotesOnHost),
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

func (c GitHubClient) UploadAssetsFile(ctx context.Context, settings config.ReleaseSettings, f *os.File, releaseID int64) error {
	// TODO(bep) retryable errors.
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

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("github: unexpected status code: %d", resp.StatusCode)
	}
	return nil

}

// Fake client is only used in tests.
type FakeClient struct {
	releaseID int64
}

func (c *FakeClient) Release(ctx context.Context, tagName string, committish string, settings config.ReleaseSettings) (int64, error) {
	c.releaseID = rand.Int63()
	return c.releaseID, nil
}

func (c *FakeClient) UploadAssetsFile(ctx context.Context, settings config.ReleaseSettings, f *os.File, releaseID int64) error {
	if c.releaseID != releaseID {
		return fmt.Errorf("fake: releaseID mismatch: %d != %d", c.releaseID, releaseID)
	}
	if f == nil {
		return fmt.Errorf("fake: nil file")
	}
	return nil
}
