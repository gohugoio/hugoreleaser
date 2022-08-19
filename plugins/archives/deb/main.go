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
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/bep/hugoreleaser/internal/config"
	"github.com/bep/hugoreleaser/pkg/model"
	"github.com/bep/hugoreleaser/pkg/plugins"
	"github.com/bep/hugoreleaser/pkg/plugins/archiveplugin"
	"github.com/goreleaser/nfpm/v2"
	_ "github.com/goreleaser/nfpm/v2/deb" // init format
	"github.com/goreleaser/nfpm/v2/files"
)

const name = "deb"

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

// Meta is fetched from archive_settings.meta in the archive configuration.
type Meta struct {
	Vendor      string
	Homepage    string
	Maintainer  string
	Description string
	License     string
}

type debArchivist struct {
	out   io.WriteCloser
	files files.Contents
	cfg   config.ArchiveSettings
	meta  Meta
}

func (a *debArchivist) Add(sourceFilename, targetPath string) error {
	a.files = append(a.files, &files.Content{
		Source:      filepath.ToSlash(sourceFilename),
		Destination: targetPath,
		FileInfo: &files.ContentFileInfo{
			Mode: 0o755,
		},
	})

	return nil
}

func (a *debArchivist) Finalize() error {
	meta := a.meta

	info := &nfpm.Info{
		Platform: "linux",
		Name:     "TODO",
		Version:  "TODO",
		/*Arch:            "TODO",



		Section:         "TODO",
		Priority:        "TODO",
		Epoch:           "TODO",
		Release:         "TODO",
		Prerelease:      "TODO",
		VersionMetadata: "TODO",*/
		Maintainer:  meta.Maintainer,
		Description: meta.Description,
		Vendor:      meta.Vendor,
		Homepage:    meta.Homepage,
		License:     meta.License,
		Overridables: nfpm.Overridables{
			Contents: a.files,
		},
	}

	packager, err := nfpm.Get("deb")
	if err != nil {
		return err
	}

	info = nfpm.WithDefaults(info)

	return packager.Package(info, a.out)
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

	settings := req.Settings

	meta, err := model.FromMap[Meta](settings.Meta)
	if err != nil {
		return err
	}

	archivist := &debArchivist{
		out:  f,
		meta: meta,
	}

	for _, file := range req.Files {
		archivist.Add(file.SourcePathAbs, file.TargetPath)
	}

	return archivist.Finalize()
}

func errResponse(err error) archiveplugin.Response {
	return archiveplugin.Response{Error: model.NewBasicError(name, err)}
}
