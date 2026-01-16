package releases

import (
	"context"
	"os"

	"github.com/gohugoio/hugoreleaser/internal/config"
)

type ReleaseInfo struct {
	Project   string
	Tag       string
	Commitish string
	Settings  config.ReleaseSettings
}

type Client interface {
	Release(ctx context.Context, info ReleaseInfo) (int64, error)
	UploadAssetsFile(ctx context.Context, info ReleaseInfo, f *os.File, releaseID int64) error
}

// PublishClient extends Client with publish-specific operations.
type PublishClient interface {
	Client

	// GetReleaseByTag retrieves a release by its tag name.
	// Returns the release ID, draft status, and error.
	GetReleaseByTag(ctx context.Context, owner, repo, tag string) (releaseID int64, isDraft bool, err error)

	// PublishRelease sets a release from draft to published.
	PublishRelease(ctx context.Context, owner, repo string, releaseID int64) error

	// UpdateFileInRepo creates or updates a file in a repository.
	// Returns the commit SHA on success.
	UpdateFileInRepo(ctx context.Context, owner, repo, path, message string, content []byte) (string, error)
}
