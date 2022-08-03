package archives

import (
	"io"
	"path/filepath"

	"github.com/bep/hugoreleaser/internal/common/ioh"
	"github.com/bep/hugoreleaser/internal/config"
	"github.com/goreleaser/nfpm/v2"
	_ "github.com/goreleaser/nfpm/v2/deb" // init format
	"github.com/goreleaser/nfpm/v2/files"
)

var _ Archiver = &ArchiveDeb{}

func newDeb(cfg config.ArchiveSettings, out io.WriteCloser) *ArchiveDeb {
	archive := &ArchiveDeb{
		out:    out,
		cfg:    cfg,
		binDir: "/usr/bin", // TODO(bep)
	}

	return archive
}

type ArchiveDeb struct {
	out    io.WriteCloser
	files  files.Contents
	binDir string
	cfg    config.ArchiveSettings
}

func (a *ArchiveDeb) AddAndClose(name string, f ioh.File) error {
	defer f.Close()

	src := f.Name()
	dst := filepath.Join(a.binDir, name)

	a.files = append(a.files, &files.Content{
		Source:      filepath.ToSlash(src),
		Destination: filepath.ToSlash(dst),
		FileInfo: &files.ContentFileInfo{
			Mode: 0o755,
		},
	})

	return nil
}

func (a *ArchiveDeb) Finalize() error {
	defer a.out.Close()

	meta := a.cfg.Meta

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
