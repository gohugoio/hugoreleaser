package ioh

import (
	"io"
	"io/fs"
)

type File interface {
	fs.File
	io.Writer
	Name() string
}

type FileInfo interface {
	ReadSeekCloser
	Name() string
}

type ReadSeekCloser interface {
	ReadSeeker
	io.Closer
}

// ReadSeeker wraps io.Reader and io.Seeker.
type ReadSeeker interface {
	io.Reader
	io.Seeker
}
