package releases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/bep/workers"
)

// CreateChecksumLines writes the SHA256 checksums as lowercase hex digits followed by
// two spaces and then the base of filename and returns a sorted slice.
func CreateChecksumLines(w *workers.Workforce, filenames ...string) ([]string, error) {
	var mu sync.Mutex
	var result []string

	r, _ := w.Start(context.Background())

	createChecksum := func(filename string) (string, error) {
		f, err := os.Open(filename)
		if err != nil {
			return "", err
		}
		defer f.Close()
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}
		return hex.EncodeToString(h.Sum(nil)), nil

	}

	r.Run(func() error {
		for _, filename := range filenames {
			checksum, err := createChecksum(filename)
			if err != nil {
				return err
			}
			mu.Lock()
			result = append(result, checksum+"  "+filepath.Base(filename))
			mu.Unlock()
		}

		return nil
	})

	if err := r.Wait(); err != nil {
		return nil, err
	}

	sort.Strings(result)

	return result, nil
}
