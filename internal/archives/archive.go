package archives

import (
	"io"

	"github.com/bep/hugoreleaser/internal/archives/archiveformats"
	"github.com/bep/hugoreleaser/internal/common/ioh"
	"github.com/bep/hugoreleaser/internal/config"
)

func New(settings config.ArchiveSettings, out io.WriteCloser) Archiver {
	switch settings.Formati {
	case archiveformats.TarGz:
		return newTarGz(out)
	case archiveformats.Zip:
		return newZip(out)
	case archiveformats.Deb:
		return newDeb(settings, out)
	default:
		panic("unsupported format")
	}
}

type Archiver interface {
	// AddAndClose adds a file to the archive, then closes it.
	// The name is the short name of the file, not the full path.
	AddAndClose(name string, f ioh.File) error

	// Finalize finalizes the archive and closes all writers in use.
	// It is not safe to call AddAndClose after Finalize.
	Finalize() error
}
