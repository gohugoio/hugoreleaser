package releases

import (
	"context"
	"os"

	"github.com/gohugoio/hugoreleaser/internal/config"
)

type ReleaseInfo struct {
	Tag       string
	Commitish string
	Settings  config.ReleaseSettings
}

type Client interface {
	Release(ctx context.Context, info ReleaseInfo) (int64, error)
	UploadAssetsFile(ctx context.Context, info ReleaseInfo, f *os.File, releaseID int64) error
}
