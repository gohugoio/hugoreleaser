package archiveplugin

import (
	"fmt"

	"github.com/bep/hugoreleaser/pkg/model"
)

var (
	_ model.Initializer = (*ArchiveFile)(nil)
	_ model.Initializer = (*Request)(nil)
)

// Request is what is sent to an external archive tool.
type Request struct {
	// Heartbeat is a string that is echoed back to the caller,
	// used to test that plugin servers are up and running.
	Heartbeat string `toml:"heartbeat"`

	// Information about the build to archive.
	// TODO(bep) needed?
	model.BuildContext

	Files []ArchiveFile `toml:"files"`

	// Filename with extension.
	OutFilename string `toml:"out_filename"`
}

// HeartbeatResponse returns a Response that, if the second return value is true,
// will be returned to the caller.
func (r Request) HeartbeatResponse() (Response, bool) {
	if r.Heartbeat == "" {
		return Response{}, false
	}
	return Response{Heartbeat: r.Heartbeat}, true
}

func (r *Request) Init() error {
	what := "archive_request"
	if r.OutFilename == "" {
		return fmt.Errorf("%s: archive request has no output filename", what)
	}
	for i := range r.Files {
		f := &r.Files[i]
		if err := f.Init(); err != nil {
			return fmt.Errorf("%s: %v", what, err)
		}
	}
	return nil
}

// Response is what is sent back from an external archive tool.
type Response struct {
	// Heartbeat is a string that is echoed back to the caller,
	// used to test that plugin servers are up and running.
	Heartbeat string `toml:"heartbeat"`

	Error *model.BasicError `toml:"err"`
}

func (r Response) Err() error {
	if r.Error == nil {
		// Make sure that resp.Err() == nil.
		return nil
	}
	return r.Error
}

type ArchiveFile struct {
	// The source filename.
	SourcePathAbs string `toml:"source_path_abs"`

	// Relative target path, including the name of the file.
	TargetPath string `toml:"target_path"`
}

func (a *ArchiveFile) Init() error {
	return nil
}
