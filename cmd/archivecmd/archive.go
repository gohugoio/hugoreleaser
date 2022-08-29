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

package archivecmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/gohugoio/hugoreleaser-plugins-api/archiveplugin"
	"github.com/gohugoio/hugoreleaser/cmd/corecmd"
	"github.com/gohugoio/hugoreleaser/internal/archives"
	"github.com/gohugoio/hugoreleaser/internal/config"
	"github.com/gohugoio/hugoreleaser/internal/plugins"

	"github.com/bep/logg"
	"github.com/gohugoio/hugoreleaser-plugins-api/model"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const commandName = "archive"

// New returns a usable ffcli.Command for the archive subcommand.
func New(core *corecmd.Core) *ffcli.Command {
	fs := flag.NewFlagSet(corecmd.CommandName+" "+commandName, flag.ExitOnError)

	archivist := NewArchivist(core)

	core.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       "archive",
		ShortUsage: corecmd.CommandName + " archive [flags] <action>",
		ShortHelp:  "Build archives from binaries and any extra files configured.",
		FlagSet:    fs,
		Exec:       archivist.Exec,
	}
}

type Archivist struct {
	infoLog logg.LevelLogger
	core    *corecmd.Core
}

// NewArchivist returns a new Archivist.
func NewArchivist(core *corecmd.Core) *Archivist {
	return &Archivist{
		core: core,
	}

}

func (b *Archivist) Init() error {
	b.infoLog = b.core.InfoLog.WithField("cmd", commandName)
	c := b.core

	startAndRegister := func(p config.Plugin) error {
		if p.IsZero() {
			return nil
		}
		if _, found := c.PluginsRegistryArchive[p.ID]; found {
			// Already started.
			return nil
		}
		infoCtx := c.InfoLog.WithField("plugin", p.ID)
		client, err := plugins.StartArchivePlugin(c.InfoLog, c.Config.GoSettings, p)
		if err != nil {
			return fmt.Errorf("error starting archive plugin %q: %w", p.ID, err)
		}

		// Send a heartbeat to the plugin to make sure it's alive.
		heartbeat := fmt.Sprintf("heartbeat-%s", time.Now())
		resp, err := client.Execute(archiveplugin.Request{Heartbeat: heartbeat})
		if err != nil {
			return fmt.Errorf("error testing archive plugin %q: %w", p.ID, err)
		}
		if resp.Heartbeat != heartbeat {
			return fmt.Errorf("error testing archive plugin %q: unexpected heartbeat response", p.ID)
		}
		infoCtx.Log(logg.String("Archive plugin started and ready for use"))
		c.PluginsRegistryArchive[p.ID] = client
		return nil
	}

	if err := startAndRegister(c.Config.ArchiveSettings.Plugin); err != nil {
		return err
	}
	for _, archive := range c.Config.Archives {
		if err := startAndRegister(archive.ArchiveSettings.Plugin); err != nil {
			return err
		}
	}
	return nil
}

func (b *Archivist) Exec(ctx context.Context, args []string) error {
	if err := b.Init(); err != nil {
		return err
	}

	r, _ := b.core.Workforce.Start(ctx)

	archiveDistDir := filepath.Join(
		b.core.DistDir,
		b.core.Config.Project,
		b.core.Tag,
		b.core.DistRootArchives,
	)
	filter := b.core.PathsBuildsCompiled

	for _, archive := range b.core.Config.Archives {
		archive := archive
		for _, archPath := range archive.ArchsCompiled {
			if !filter.Match(archPath.Path) {
				continue
			}
			archPath := archPath
			archiveSettings := archive.ArchiveSettings
			arch := archPath.Arch
			buildInfo := model.BuildInfo{
				Project: b.core.Config.Project,
				Tag:     b.core.Tag,
				Goos:    arch.Os.Goos,
				Goarch:  arch.Goarch,
			}

			r.Run(func() (err error) {

				outDir := filepath.Join(archiveDistDir, filepath.FromSlash(archPath.Path))

				outFilename := filepath.Join(
					outDir,
					archPath.Name,
				)

				b.infoLog.WithField("file", outFilename).Log(logg.String("Archive"))

				if b.core.Try {
					return nil
				}

				binaryFilename := filepath.Join(
					b.core.DistDir,
					b.core.Config.Project,
					b.core.Tag,
					b.core.DistRootBuilds,
					arch.BinaryPath(),
				)

				binFi, err := os.Stat(binaryFilename)
				if err != nil {
					return fmt.Errorf("%s: binary file not found: %q", commandName, binaryFilename)
				}

				if err := os.MkdirAll(filepath.Dir(outFilename), 0o755); err != nil {
					return err
				}

				buildRequest := archiveplugin.Request{
					BuildInfo:   buildInfo,
					Settings:    archiveSettings.CustomSettings,
					OutFilename: outFilename,
				}

				buildRequest.Files = append(buildRequest.Files, archiveplugin.ArchiveFile{
					SourcePathAbs: binaryFilename,
					TargetPath:    path.Join(archiveSettings.BinaryDir, arch.BuildSettings.Binary),
					Mode:          binFi.Mode(),
				})

				for _, extraFile := range archiveSettings.ExtraFiles {
					buildRequest.Files = append(buildRequest.Files, archiveplugin.ArchiveFile{
						SourcePathAbs: filepath.Join(b.core.ProjectDir, extraFile.SourcePath),
						TargetPath:    extraFile.TargetPath,
						Mode:          extraFile.Mode,
					})
				}

				err = archives.Build(
					b.core,
					b.infoLog,
					archiveSettings,
					buildRequest,
				)

				if err != nil {
					return err
				}

				return nil

			})

		}
	}

	return r.Wait()

}
