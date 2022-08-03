package archivecmd

import (
	"context"
	"flag"
	"os"
	"path/filepath"

	"github.com/bep/hugoreleaser/cmd/corecmd"
	"github.com/bep/hugoreleaser/internal/archives"
	"github.com/bep/hugoreleaser/internal/archives/archiveformats"
	"github.com/bep/hugoreleaser/internal/common/templ"
	"github.com/bep/hugoreleaser/internal/external"
	"github.com/bep/hugoreleaser/internal/external/messages"
	"github.com/bep/hugoreleaser/internal/model"
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
		archs := b.core.Config.GlobArchs(archive.PathsCompiled)

		for _, arch := range archs {
			// Capture these for the Go routine below.
			archive := archive
			arch := arch
			archiveSettings := archive.ArchiveSettings

			r.Run(func() (err error) {
				ctx := ArchiveTemplateContext{
					model.BuildContext{
						Project: b.core.Config.Project,
						Ref:     b.core.Ref,
						Goos:    arch.Os.Goos,
						Goarch:  arch.Goarch,
					},
				}

				archiveFormat := archive.ArchiveSettings.Formati
				name := templ.Sprintt(archive.ArchiveSettings.NameTemplate, ctx)
				name = archiveSettings.ReplacementsReplacer.Replace(name)

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

				if archiveFormat == archiveformats.External {
					/*message := ArchiveExternalToolRequest{
						ArchiveTemplateContext: ctx,
						BaseOutFileName:        outFilename,
					}
					fmt.Println(message)*/
					// The extension is determined by the external tool.
					b.infoLog.WithField("file", outFilename+".unknown").Log(logg.String("Archiving using external tool"))
					resp, err := external.BuildArchive(messages.ArchiveRequest{
						BuildContext:    ctx.BuildContext,
						BaseOutFilename: outFilename,
					})
					if err != nil {
						return err
					}

					outFilename += resp.Ext

					b.infoLog.WithField("file", outFilename).Log(logg.String("Archiving using external tool"))

					return nil
				}

				outFilename += "." + archiveFormat.String()

				b.infoLog.WithField("file", outFilename).Log(logg.String("Archiving"))

				if b.core.Try {
					return nil
				}

				if err := os.MkdirAll(filepath.Dir(outFilename), 0o755); err != nil {
					return err
				}

				outFile, err := os.Create(outFilename)
				if err != nil {
					return err
				}

				archiver := archives.New(archive.ArchiveSettings, outFile)
				defer func() {
					err = archiver.Finalize()
				}()

				binaryFilename := filepath.Join(
					b.core.DistDir,
					b.core.Config.Project,
					b.core.Ref,
					b.core.DistRootBuilds,
					arch.BinaryPath(),
				)

				// First add the main binary.
				entryFile, err := os.Open(binaryFilename)
				if err != nil {
					return err
				}

				if err := archiver.AddAndClose(arch.BuildSettings.Binary, entryFile); err != nil {
					return err
				}

				// Then, finally, add any extra files confgured,
				// e.g. a README.md file or a LICENSE file.
				for _, name := range archiveSettings.ExtraFiles {
					filename := filepath.Join(b.core.ProjectDir, name)
					entryFile, err := os.Open(filename)
					if err != nil {
						return err
					}
					if err := archiver.AddAndClose(name, entryFile); err != nil {
						return err
					}
				}

				return
			})
		}
	}

	return r.Wait()
}
