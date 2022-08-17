package archivecmd

import (
	"context"
	"flag"
	"os"
	"path"
	"path/filepath"

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
	archivist := &Archivist{
		core: core,
	}

	fs := flag.NewFlagSet(corecmd.CommandName+" "+commandName, flag.ExitOnError)

	core.RegisterFlags(fs)

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
}

func (b *Archivist) init() error {
	b.infoLog = b.core.InfoLog.WithField("cmd", commandName)
	return nil
}

func (b *Archivist) Exec(ctx context.Context, args []string) error {
	if err := b.init(); err != nil {
		return err
	}

	r, _ := b.core.Workforce.Start(ctx)

	err := b.core.Config.ForEachArchiveArch(nil, func(archive config.Archive, archPath config.BuildArchPath) error {
		archiveSettings := archive.ArchiveSettings
		arch := archPath.Arch

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
				b.core.DistDir,
				b.core.Config.Project,
				b.core.Tag,
				b.core.DistRootArchives,
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

			// TODO(bep) core.Try

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
