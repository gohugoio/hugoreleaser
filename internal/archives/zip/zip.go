package zip

import (
	"archive/zip"
	"io"

	"github.com/bep/hugoreleaser/internal/common/ioh"
)

func New(out io.WriteCloser) *Archive {
	archive := &Archive{
		out:  out,
		zipw: zip.NewWriter(out),
	}

	return archive
}

type Archive struct {
	out  io.WriteCloser
	zipw *zip.Writer
}

func (a *Archive) AddAndClose(targetPath string, f ioh.File) error {
	defer f.Close()

	zw, err := a.zipw.Create(targetPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(zw, f)

	return err
}

func (a *Archive) Finalize() error {
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
