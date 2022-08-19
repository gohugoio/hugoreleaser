// Copyright 2022 The Hugoreleaser Authors
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
