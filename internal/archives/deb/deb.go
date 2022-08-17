package deb

import (
	"io"
	"path/filepath"

	"github.com/bep/hugoreleaser/internal/common/ioh"
	"github.com/bep/hugoreleaser/internal/config"
	"github.com/bep/hugoreleaser/pkg/model"
	"github.com/goreleaser/nfpm/v2"
	_ "github.com/goreleaser/nfpm/v2/deb" // init format
	"github.com/goreleaser/nfpm/v2/files"
)

func New(cfg config.ArchiveSettings, out io.WriteCloser) (*Archive, error) {
	meta, err := model.FromMap[Meta](cfg.Meta)
	if err != nil {
		return nil, err
	}

	archive := &Archive{
		out:  out,
		cfg:  cfg,
		meta: meta,
	}

	return archive, nil
}

// Meta is fetched from archive_settings.meta in the archive configuration.
type Meta struct {
	Vendor      string
	Homepage    string
	Maintainer  string
	Description string
	License     string
}

type Archive struct {
	out   io.WriteCloser
	files files.Contents
	cfg   config.ArchiveSettings
	meta  Meta
}

func (a *Archive) AddAndClose(targetPath string, f ioh.File) error {
	defer f.Close()
	src := f.Name()

	a.files = append(a.files, &files.Content{
		Source:      filepath.ToSlash(src),
		Destination: targetPath,
		FileInfo: &files.ContentFileInfo{
			Mode: 0o755,
		},
	})

	return nil
}

func (a *Archive) Finalize() error {
	defer a.out.Close()

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
