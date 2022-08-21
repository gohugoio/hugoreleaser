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

package renamer

import (
	"io"
	"sync"

	"github.com/gohugoio/hugoreleaser/internal/common/ioh"
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
