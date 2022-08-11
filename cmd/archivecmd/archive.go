package archivecmd

import (
	"context"
	"flag"
	"os"
	"path"
	"path/filepath"

	"github.com/bep/hugoreleaser/cmd/corecmd"
	"github.com/bep/hugoreleaser/internal/common/templ"
	"github.com/bep/hugoreleaser/internal/external"
	"github.com/bep/hugoreleaser/pkg/plugins/archiveplugin"

	"github.com/bep/hugoreleaser/pkg/model"
	"github.com/bep/logg"
	"github.com/peterbourgon/ff/v3/ffcli"
)

// New returns a usable ffcli.Command for the archive subcommand.
func New(core *corecmd.Core) *ffcli.Command {
	archivist := &Archivist{
		core: core,
	}

	fs := flag.NewFlagSet(corecmd.CommandName+" archive", flag.ExitOnError)

	core.RegisterFlags(fs)
	core.AddInitDoneListener(func() {
		archivist.infoLog = core.InfoLog.WithField("cmd", "archive")
	})

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

func (b *Archivist) Exec(ctx context.Context, args []string) error {
	r, _ := b.core.Workforce.Start(ctx)
	for _, archive := range b.core.Config.Archives {
		archive := archive
		archs := b.core.Config.GlobArchs(archive.PathsCompiled)
		archiveSettings := archive.ArchiveSettings

		for _, arch := range archs {
			// Capture these for the Go routine below.
			archive := archive
			arch := arch

			r.Run(func() (err error) {
				ctx := ArchiveTemplateContext{
					model.BuildContext{
						Project: b.core.Config.Project,
						Ref:     b.core.Ref,
						Goos:    arch.Os.Goos,
						Goarch:  arch.Goarch,
					},
				}

				name := templ.Sprintt(archive.ArchiveSettings.NameTemplate, ctx)
				name = archiveSettings.ReplacementsCompiled.Replace(name)

				outFilename := filepath.Join(
					b.core.DistDir,
					b.core.Config.Project,
					b.core.Ref,
					b.core.DistRootArchives,
					arch.Build.Path,
					arch.Os.Goos,
					arch.Goarch,
					name,
				)

				outFilename += archiveSettings.Type.Extension
				binaryFilename := filepath.Join(
					b.core.DistDir,
					b.core.Config.Project,
					b.core.Ref,
					b.core.DistRootBuilds,
					arch.BinaryPath(),
				)

				if err := os.MkdirAll(filepath.Dir(outFilename), 0o755); err != nil {
					return err
				}

				// TODO(bep) core.Try

				b.infoLog.WithField("file", outFilename).Log(logg.String("Archiving"))

				buildRequest := archiveplugin.Request{
					BuildContext: ctx.BuildContext,
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

				if b.core.Try {
					return nil
				}

				err = external.BuildArchive(
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
