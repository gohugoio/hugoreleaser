package archivecmd

import (
	"context"
	"flag"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/bep/hugoreleaser/cmd/corecmd"
	"github.com/bep/hugoreleaser/internal/archives"
	"github.com/bep/hugoreleaser/internal/common/templ"
	"github.com/bep/hugoreleaser/internal/config"
	"github.com/bep/hugoreleaser/pkg/plugins/archiveplugin"

	"github.com/bep/hugoreleaser/pkg/model"
	"github.com/bep/logg"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const commandName = "archive"

// New returns a usable ffcli.Command for the archive subcommand.
func New(core *corecmd.Core) *ffcli.Command {
	fs := flag.NewFlagSet(corecmd.CommandName+" "+commandName, flag.ExitOnError)
	core.RegisterFlags(fs)
	archivist := NewArchivist(core, fs)

	return &ffcli.Command{
		Name:       "archive",
		ShortUsage: corecmd.CommandName + " archive [flags] <action>",
		ShortHelp:  "TODO(bep)",
		FlagSet:    fs,
		Exec:       archivist.Exec,
	}
}

type ArchiveTemplateContext struct {
	model.BuildContext
}

type Archivist struct {
	infoLog logg.LevelLogger
	core    *corecmd.Core

	initOnce sync.Once
	initErr  error
}

// NewArchivist returns a new Archivist.
func NewArchivist(core *corecmd.Core, fs *flag.FlagSet) *Archivist {
	return &Archivist{
		core: core,
	}
}

func (b *Archivist) Init() error {
	b.initOnce.Do(func() {
		b.infoLog = b.core.InfoLog.WithField("cmd", commandName)
	})
	return b.initErr
}

func (b *Archivist) Exec(ctx context.Context, args []string) error {
	if err := b.Init(); err != nil {
		return err
	}

	r, _ := b.core.Workforce.Start(ctx)

	err := b.core.Config.ForEachArchiveArch(nil, func(archive config.Archive, archPath config.BuildArchPath) error {
		archiveSettings := archive.ArchiveSettings
		arch := archPath.Arch
		archiveDistDir := filepath.Join(
			b.core.DistDir,
			b.core.Config.Project,
			b.core.Tag,
			b.core.DistRootArchives,
		)

		// Remove and recreate the archive dist dir.
		// We do it on this level to allow adding artifacts between the archive and release steps.
		_ = os.RemoveAll(archiveDistDir)
		if err := os.MkdirAll(archiveDistDir, 0o755); err != nil {
			return err
		}

		r.Run(func() (err error) {
			archiveTemplCtx := ArchiveTemplateContext{
				model.BuildContext{
					Project: b.core.Config.Project,
					Tag:     b.core.Tag,
					Goos:    arch.Os.Goos,
					Goarch:  arch.Goarch,
				},
			}

			name := templ.Sprintt(archive.ArchiveSettings.NameTemplate, archiveTemplCtx)
			name = archiveSettings.ReplacementsCompiled.Replace(name)

			outFilename := filepath.Join(
				archiveDistDir,
				filepath.FromSlash(archPath.Path),
				name,
			)

			outFilename += archiveSettings.Type.Extension
			binaryFilename := filepath.Join(
				b.core.DistDir,
				b.core.Config.Project,
				b.core.Tag,
				b.core.DistRootBuilds,
				arch.BinaryPath(),
			)

			if err := os.MkdirAll(filepath.Dir(outFilename), 0o755); err != nil {
				return err
			}

			b.infoLog.WithField("file", outFilename).Log(logg.String("Archiving"))

			buildRequest := archiveplugin.Request{
				BuildContext: archiveTemplCtx.BuildContext,
				OutFilename:  outFilename,
			}

			buildRequest.Files = append(buildRequest.Files, archiveplugin.ArchiveFile{
				SourcePathAbs: binaryFilename,
				TargetPath:    path.Join(archiveSettings.BinaryDir, arch.BuildSettings.Binary),
			})

			for _, extraFile := range archiveSettings.ExtraFiles {
				buildRequest.Files = append(buildRequest.Files, archiveplugin.ArchiveFile{
					// TODO(bep) unify slashes.
					SourcePathAbs: filepath.Join(b.core.ProjectDir, extraFile.SourcePath),
					TargetPath:    extraFile.TargetPath,
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
		return nil
	})

	errWait := r.Wait()

	if err != nil {
		return err
	}

	return errWait
}
