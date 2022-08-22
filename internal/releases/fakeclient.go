package releases

import (
	"context"
	"fmt"
	"math/rand"
	"os"
)

// Fake client is only used in tests.
type FakeClient struct {
	releaseID int64
}

func (c *FakeClient) Release(ctx context.Context, info ReleaseInfo) (int64, error) {
	// Tests depend on this string.
	fmt.Printf("fake: release: %#v\n", info)
	if info.Settings.ReleaseNotesFilename != "" {
		_, err := os.Stat(info.Settings.ReleaseNotesFilename)
		if err != nil {
			return 0, err
		}
	}
	c.releaseID = rand.Int63()
	return c.releaseID, nil
}

func (c *FakeClient) UploadAssetsFile(ctx context.Context, info ReleaseInfo, f *os.File, releaseID int64) error {
	if c.releaseID != releaseID {
		return fmt.Errorf("fake: releaseID mismatch: %d != %d", c.releaseID, releaseID)
	}
	if f == nil {
		return fmt.Errorf("fake: nil file")
	}
	return nil
}
