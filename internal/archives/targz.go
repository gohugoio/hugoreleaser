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

	gw, _ := gzip.NewWriterLevel(out, gzip.BestCompression)
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

func (a *ArchiveTarGz) AddAndClose(targetPath string, f ioh.File) error {
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "") // TODO(bep) symlink handling?
	if err != nil {
		return err
	}
	header.Name = targetPath

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
	if err := a.tw.Close(); err != nil {
		return err
	}
	if err := a.gw.Close(); err != nil {
		return err
	}
	return a.out.Close()

}
