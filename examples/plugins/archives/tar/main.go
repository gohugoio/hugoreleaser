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

package main

import (
	"archive/tar"
	"io"
	"log"
	"os"

	"github.com/bep/hugoreleaser/pkg/model"
	"github.com/bep/hugoreleaser/pkg/plugins"
	"github.com/bep/hugoreleaser/pkg/plugins/archiveplugin"
)

const name = "tarplugin"

func main() {
	server, err := plugins.NewServer(
		func(d plugins.Dispatcher, req archiveplugin.Request) archiveplugin.Response {
			d.Infof("Creating archive %s", req.OutFilename)
			if err := createArchive(req); err != nil {
				return errResponse(err)
			}
			// Empty response is a success.
			return archiveplugin.Response{}
		},
	)
	if err != nil {
		log.Fatalf("Failed to create server: %s", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %s", err)
	}

	_ = server.Wait()
}

func createArchive(req archiveplugin.Request) error {
	if err := req.Init(); err != nil {
		return err
	}

	f, err := os.Create(req.OutFilename)
	if err != nil {
		return err
	}
	defer f.Close()

	tarw := tar.NewWriter(f)
	defer tarw.Close()

	for _, file := range req.Files {
		filename := file.SourcePathAbs
		info, err := os.Stat(filename)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		header.Name = file.TargetPath

		if err := tarw.WriteHeader(header); err != nil {
			return err
		}

		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tarw, f)
		if err != nil {
			return err
		}
	}

	return nil
}

func errResponse(err error) archiveplugin.Response {
	return archiveplugin.Response{Error: model.NewBasicError(name, err)}
}
