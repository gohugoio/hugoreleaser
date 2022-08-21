// Copyright 2022 The Hugoreleaser Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package archiveplugin

import (
	"fmt"

	"github.com/gohugoio/hugoreleaser/plugins"
	"github.com/gohugoio/hugoreleaser/plugins/model"
)

const (
	// Version gets incremented on incompatible changes to the archive plugin or its runtime,
	// think of this as a major version increment in semver terms.
	// This should almost never happen, but if it does, the old archive plugin will probably not work as expected.
	// This will be detected on Hugoreleaser startup and the build will fail.
	// The plugin server then needs to be updated and re-tested.
	Version = 0
)

var (
	_ model.Initializer = (*ArchiveFile)(nil)
	_ model.Initializer = (*Request)(nil)
)

// Request is what is sent to an external archive tool.
type Request struct {
	// Version is the archive plugin version.
	// This is just used for validation on startup.
	Version int `toml:"version"`

	// Heartbeat is a string that is echoed back to the caller,
	// used to test that plugin servers are up and running.
	Heartbeat string `toml:"heartbeat"`

	// BuildContext holds the basic build information about the current build.
	BuildContext model.BuildContext `toml:"build_context"`

	// Settings for the archive.
	// This is the content of archive_settings.custom_settings.
	Settings map[string]any `toml:"settings"`

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
	var err *plugins.Error
	if r.Version != Version {
		err = &plugins.Error{Msg: fmt.Sprintf("archive plugin version mismatch: client sent %d, server is at %d", r.Version, Version)}
	}
	return Response{Heartbeat: r.Heartbeat, Error: err}, true
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

	Error *plugins.Error `toml:"err"`
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
