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

package archives

import (
	"fmt"
	"io"

	"github.com/bep/hugoreleaser/internal/archives/archiveformats"
	"github.com/bep/hugoreleaser/internal/archives/renamer"
	"github.com/bep/hugoreleaser/internal/archives/targz"
	"github.com/bep/hugoreleaser/internal/archives/zip"
	"github.com/bep/hugoreleaser/internal/common/ioh"
	"github.com/bep/hugoreleaser/internal/config"
)

func New(settings config.ArchiveSettings, out io.WriteCloser) (Archiver, error) {
	switch settings.Type.FormatParsed {
	case archiveformats.TarGz:
		return targz.New(out), nil
	case archiveformats.Zip:
		return zip.New(out), nil
	case archiveformats.Rename:
		return renamer.New(out), nil
	default:
		return nil, fmt.Errorf("unsupported archive format %q", settings.Type.Format)
	}
}

type Archiver interface {
	// AddAndClose adds a file to the archive, then closes it.
	AddAndClose(dir string, f ioh.File) error

	// Finalize finalizes the archive and closes all writers in use.
	// It is not safe to call AddAndClose after Finalize.
	Finalize() error
}
