package archives

import (
	"archive/tar"
	"compress/gzip"
	"io"

	"github.com/bep/hugoreleaser/internal/common/ioh"
)

var _ Archiver = &ArchiveTarGz{}

func newTarGz(out io.WriteCloser) *ArchiveTarGz {
	archive := &ArchiveTarGz{
		out: out,
	}

	gw := gzip.NewWriter(out)
	tw := tar.NewWriter(gw)

	archive.gw = gw
	archive.tw = tw

	return archive
}

type ArchiveTarGz struct {
	out io.WriteCloser
	gw  *gzip.Writer
	tw  *tar.Writer
}

func (a *ArchiveTarGz) AddAndClose(name string, f ioh.File) error {
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	// TODO(bep) check second argument vs symlinks.
	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return err
	}

	// Use full path as name to preserve structure.
	header.Name = f.Name()

	err = a.tw.WriteHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(a.tw, f)
	if err != nil {
		return err
	}

	return nil
}

func (a *ArchiveTarGz) Finalize() error {
	err1 := a.gw.Close()
	err2 := a.out.Close()

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}

	return nil
}
