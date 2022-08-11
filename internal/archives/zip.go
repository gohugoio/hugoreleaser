package archives

import (
	"archive/zip"
	"io"

	"github.com/bep/hugoreleaser/internal/common/ioh"
)

var _ Archiver = &ArchiveTarGz{}

func newZip(out io.WriteCloser) *ArchiveZip {
	archive := &ArchiveZip{
		out:  out,
		zipw: zip.NewWriter(out),
	}

	return archive
}

type ArchiveZip struct {
	out  io.WriteCloser
	zipw *zip.Writer
}

func (a *ArchiveZip) AddAndClose(targetPath string, f ioh.File) error {
	defer f.Close()

	zw, err := a.zipw.Create(targetPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(zw, f)

	return err
}

func (a *ArchiveZip) Finalize() error {
	err1 := a.zipw.Close()
	err2 := a.out.Close()

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}

	return nil
}
