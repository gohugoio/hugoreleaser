package renamer

import (
	"io"
	"sync"

	"github.com/bep/hugoreleaser/internal/common/ioh"
)

// New returns a new Archiver for the given writer.
func New(out io.WriteCloser) *Renamer {
	archive := &Renamer{
		out: out,
	}
	return archive
}

// Renamer is an Archiver that just writes the first File received in AddAndClose to the underlying writer,
// and drops the rest.
// This construct is most useful for testing.
type Renamer struct {
	out io.WriteCloser

	writeOnce sync.Once
}

func (a *Renamer) AddAndClose(targetPath string, f ioh.File) error {
	var err error
	a.writeOnce.Do(func() {
		_, err = io.Copy(a.out, f)
	})
	return err
}

func (a *Renamer) Finalize() error {
	return a.out.Close()
}
