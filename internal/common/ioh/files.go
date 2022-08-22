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

package ioh

import (
	"io"
	"io/fs"
	"os"
)

type File interface {
	fs.File
	io.Writer
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

// RemoveAllMkdirAll is a wrapper for os.RemoveAll and os.MkdirAll.
func RemoveAllMkdirAll(dirname string) error {
	_ = os.RemoveAll(dirname)
	return os.MkdirAll(dirname, 0o755)
}
