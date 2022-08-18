package archives

import (
	"fmt"
	"io"

	"github.com/bep/hugoreleaser/internal/archives/archiveformats"
	"github.com/bep/hugoreleaser/internal/archives/deb"
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
	case archiveformats.Deb:
		return deb.New(settings, out)
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
