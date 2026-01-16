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
	if info.Settings.ReleaseNotesSettings.Filename != "" {
		_, err := os.Stat(info.Settings.ReleaseNotesSettings.Filename)
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

// Ensure FakeClient implements PublishClient.
var _ PublishClient = &FakeClient{}

func (c *FakeClient) GetReleaseByTag(ctx context.Context, owner, repo, tag string) (int64, bool, error) {
	fmt.Printf("fake: GetReleaseByTag: owner=%s repo=%s tag=%s\n", owner, repo, tag)
	c.releaseID = rand.Int63()
	return c.releaseID, true, nil // Return as draft for testing.
}

func (c *FakeClient) PublishRelease(ctx context.Context, owner, repo string, releaseID int64) error {
	fmt.Printf("fake: PublishRelease: owner=%s repo=%s releaseID=%d\n", owner, repo, releaseID)
	return nil
}

func (c *FakeClient) UpdateFileInRepo(ctx context.Context, owner, repo, path, message string, content []byte) (string, error) {
	fmt.Printf("fake: UpdateFileInRepo: owner=%s repo=%s path=%s message=%q\n", owner, repo, path, message)
	return "fakesha123", nil
}
