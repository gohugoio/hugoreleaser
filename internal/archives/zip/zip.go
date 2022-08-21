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

package zip

import (
	"archive/zip"
	"io"

	"github.com/gohugoio/hugoreleaser/internal/common/ioh"
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
