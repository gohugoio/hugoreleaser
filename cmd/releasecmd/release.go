package releasecmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gobwas/glob"

	"github.com/bep/hugoreleaser/cmd/corecmd"
	"github.com/bep/hugoreleaser/internal/common/matchers"
	"github.com/bep/hugoreleaser/internal/config"
	"github.com/bep/hugoreleaser/internal/releases"
	"github.com/bep/logg"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const commandName = "release"

// New returns a usable ffcli.Command for the release subcommand.
func New(core *corecmd.Core) *ffcli.Command {

	fs := flag.NewFlagSet(corecmd.CommandName+" "+commandName, flag.ExitOnError)

	releaser := NewReleaser(core, fs)

	core.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       "release",
		ShortUsage: corecmd.CommandName + " build [flags] <action>",
		ShortHelp:  "TODO(bep)",
		FlagSet:    fs,
		Exec:       releaser.Exec,
	}
}

// NewReleaser returns a new Releaser.
func NewReleaser(core *corecmd.Core, fs *flag.FlagSet) *Releaser {
	r := &Releaser{
		core: core,
	}

	fs.StringVar(&r.commitish, "commitish", "", "The commitish value that determines where the Git tag is created from.")
	fs.StringVar(&r.paths, "paths", "/archives/**", "The archives to release (defaults to all).")

	return r
}

type Releaser struct {
	core    *corecmd.Core
	infoLog logg.LevelLogger

	// Flags
	commitish string
	paths     string

	pathsCompiled matchers.Matcher

	initOnce sync.Once
	initErr  error
}

func (b *Releaser) Init() error {
	b.initOnce.Do(func() {
		if b.commitish == "" {
			b.initErr = fmt.Errorf("%s: flag -commitish is required", commandName)
			return
		}

		const prefix = "/archives/"

		if !strings.HasPrefix(b.paths, prefix) {
			b.initErr = fmt.Errorf("%s: flag -paths must start with %s", commandName, prefix)
			return
		}

		// Strip the /archives/ prefix. We currently don't use that,
		// it's just there to make the config easier to understand.
		paths := strings.TrimPrefix(b.paths, prefix)

		var err error
		b.pathsCompiled, err = glob.Compile(paths)
		if err != nil {
			b.initErr = fmt.Errorf("%s: invalid -paths value: %s", commandName, err)
			return
		}

		b.infoLog = b.core.InfoLog.WithField("cmd", commandName)
	})
	return b.initErr

}

func (b *Releaser) Exec(ctx context.Context, args []string) error {
	if err := b.Init(); err != nil {
		return err
	}

	if len(b.core.Config.Releases) == 0 {
		return fmt.Errorf("%s: no releases defined in config", commandName)
	}

	logCtx := b.infoLog.WithField("matching", b.paths)

	releaseMatches := b.core.Config.FindReleases(b.pathsCompiled)
	if len(releaseMatches) == 0 {
		return fmt.Errorf("%s: no releases found matching %s", commandName, b.paths)
	}

	for _, release := range releaseMatches {

		releaseDir := filepath.Join(
			b.core.DistDir,
			b.core.Config.Project,
			b.core.Tag,
			b.core.DistRootReleases,
			filepath.FromSlash(release.Dir),
		)

		if _, err := os.Stat(releaseDir); err == nil || os.IsNotExist(err) {
			if !os.IsNotExist(err) {
				// Start fresh.
				if err := os.RemoveAll(releaseDir); err != nil {
					return fmt.Errorf("%s: failed to remove release directory %q: %s", commandName, releaseDir, err)
				}
			}
			if err := os.MkdirAll(releaseDir, 0755); err != nil {
				return fmt.Errorf("%s: failed to create release directory %q: %s", commandName, releaseDir, err)
			}

		}

		// First collect all files to be released.
		var archiveFilenames []string

		err := b.core.Config.ForEachArchiveArch(
			b.pathsCompiled,
			func(archive config.Archive, archPath config.BuildArchPath) error {

				archiveDir := filepath.Join(
					b.core.DistDir,
					b.core.Config.Project,
					b.core.Tag,
					b.core.DistRootArchives,
					filepath.FromSlash(archPath.Path),
				)

				entries, err := os.ReadDir(archiveDir)
				if err != nil {
					return err
				}

				for _, entry := range entries {
					if entry.IsDir() {
						continue
					}
					archiveFilenames = append(archiveFilenames, filepath.Join(archiveDir, entry.Name()))
				}

				return nil

			})

		if err != nil {
			return err
		}

		if len(archiveFilenames) == 0 {
			return fmt.Errorf("%s: no files found for release %q", commandName, release.Dir)
		}

		// Create a checksum.txt file.
		checksumLines, err := releases.CreateChecksumLines(b.core.Workforce, archiveFilenames...)
		if err != nil {
			return err
		}
		checksumFilename := filepath.Join(releaseDir, "checksum.txt")
		err = func() error {
			f, err := os.Create(checksumFilename)
			if err != nil {
				return err
			}
			defer f.Close()

			for _, line := range checksumLines {
				_, err := f.WriteString(line + "\n")
				if err != nil {
					return err
				}
			}

			return nil
		}()

		if err != nil {
			return fmt.Errorf("%s: failed to create checksum file %q: %s", commandName, checksumFilename, err)
		}

		archiveFilenames = append(archiveFilenames, checksumFilename)

		logCtx.Log(logg.String(fmt.Sprintf("Prepared %d files to archive: %v", len(archiveFilenames), archiveFilenames)))

		// Now create the release archive and upload files.
		client, err := releases.NewClient(ctx, release.ReleaseSettings.TypeParsed)
		if err != nil {
			return fmt.Errorf("%s: failed to create release client: %v", commandName, err)
		}
		if b.core.Try {
			client = &releases.FakeClient{}
		}

		releaseID, err := client.Release(ctx, b.core.Tag, b.commitish, release.ReleaseSettings)
		if err != nil {
			return fmt.Errorf("%s: failed to create release: %v", commandName, err)
		}
		r, ctx := b.core.Workforce.Start(ctx)

		for _, archiveFilename := range archiveFilenames {
			archiveFilename := archiveFilename
			r.Run(func() error {
				f, err := os.Open(archiveFilename)
				if err != nil {
					return err
				}
				defer f.Close()
				logCtx.Log(logg.String(fmt.Sprintf("Uploading release file %s", archiveFilename)))
				if err := client.UploadAssetsFile(ctx, release.ReleaseSettings, f, releaseID); err != nil {
					return err
				}
				return nil

			})
		}

		if err := r.Wait(); err != nil {
			return fmt.Errorf("%s: failed to upload files: %v", commandName, err)
		}

	}

	return nil

}
