package archives

import (
	"io"

	"github.com/bep/hugoreleaser/internal/archives/archiveformats"
	"github.com/bep/hugoreleaser/internal/common/ioh"
	"github.com/bep/hugoreleaser/internal/config"
)

func New(settings config.ArchiveSettings, out io.WriteCloser) (Archiver, error) {
	switch settings.Type.FormatParsed {
	case archiveformats.TarGz:
		return newTarGz(out), nil
	case archiveformats.Zip:
		return newZip(out), nil
	case archiveformats.Deb:
		return newDeb(settings, out)
	default:
		panic("unsupported format")
	}
}

type Archiver interface {
	// AddAndClose adds a file to the archive, then closes it.
	AddAndClose(dir string, f ioh.File) error

	// Finalize finalizes the archive and closes all writers in use.
	// It is not safe to call AddAndClose after Finalize.
	Finalize() error
}
