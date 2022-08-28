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

package releasecmd

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/bep/logg"
	"github.com/gohugoio/hugoreleaser/cmd/corecmd"
	"github.com/gohugoio/hugoreleaser/internal/common/matchers"
	"github.com/gohugoio/hugoreleaser/internal/common/templ"
	"github.com/gohugoio/hugoreleaser/internal/config"
	"github.com/gohugoio/hugoreleaser/internal/releases"
	"github.com/gohugoio/hugoreleaser/internal/releases/changelog"
	"github.com/gohugoio/hugoreleaser/staticfiles"
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
		ShortHelp:  "Prepare and publish one or more releases.",
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

	return r
}

type Releaser struct {
	core    *corecmd.Core
	infoLog logg.LevelLogger

	// Flags
	commitish string
}

func (b *Releaser) Init() error {
	if b.commitish == "" {
		return fmt.Errorf("%s: flag -commitish is required", commandName)
	}

	b.infoLog = b.core.InfoLog.WithField("cmd", commandName)

	releaseMatches := b.core.Config.FindReleases(b.core.PathsReleasesCompiled)
	if len(releaseMatches) == 0 {
		return fmt.Errorf("%s: no releases found matching -paths %v", commandName, b.core.Paths)
	}
	for _, r := range releaseMatches {
		if err := releases.Validate(r.ReleaseSettings.TypeParsed); err != nil {
			return err
		}
	}

	return nil
}

func (b *Releaser) Exec(ctx context.Context, args []string) error {
	if err := b.Init(); err != nil {
		return err
	}

	if len(b.core.Config.Releases) == 0 {
		return fmt.Errorf("%s: no releases defined in config", commandName)
	}

	logFields := logg.Fields{
		logg.Field{
			Name: "tag", Value: b.core.Tag,
		},
		logg.Field{
			Name: "commitish", Value: b.commitish,
		},
	}

	if len(b.core.Paths) > 0 {
		logFields = append(logFields, logg.Field{Name: "paths", Value: b.core.Paths})
	}

	logCtx := b.infoLog.WithFields(logFields)

	logCtx.Log(logg.String("Finding releases"))
	releaseMatches := b.core.Config.FindReleases(b.core.PathsReleasesCompiled)

	for _, release := range releaseMatches {
		if err := b.handleRelease(ctx, logCtx, release); err != nil {
			return err
		}

	}

	return nil
}

type releaseContext struct {
	Ctx        context.Context
	Log        logg.LevelLogger
	ReleaseDir string
	Client     releases.Client
	Info       releases.ReleaseInfo
}

func (b *Releaser) handleRelease(ctx context.Context, logCtx logg.LevelLogger, release config.Release) error {
	releaseDir := filepath.Join(
		b.core.DistDir,
		b.core.Config.Project,
		b.core.Tag,
		b.core.DistRootReleases,
		filepath.FromSlash(release.Path),
	)

	info := releases.ReleaseInfo{
		Project:   b.core.Config.Project,
		Tag:       b.core.Tag,
		Commitish: b.commitish,
		Settings:  release.ReleaseSettings,
	}

	var client releases.Client
	if b.core.Try {
		client = &releases.FakeClient{}
	} else {
		var err error
		client, err = releases.NewClient(ctx, release.ReleaseSettings.TypeParsed)
		if err != nil {
			return fmt.Errorf("%s: failed to create release client: %v", commandName, err)
		}
	}

	rctx := releaseContext{
		Ctx:        ctx,
		Log:        logCtx,
		ReleaseDir: releaseDir,
		Info:       info,
		Client:     client,
	}

	if _, err := os.Stat(rctx.ReleaseDir); err == nil || os.IsNotExist(err) {
		if !os.IsNotExist(err) {
			// Start fresh.
			if err := os.RemoveAll(rctx.ReleaseDir); err != nil {
				return fmt.Errorf("%s: failed to remove release directory %q: %s", commandName, rctx.ReleaseDir, err)
			}
		}
		if err := os.MkdirAll(releaseDir, 0o755); err != nil {
			return fmt.Errorf("%s: failed to create release directory %q: %s", commandName, rctx.ReleaseDir, err)
		}

	}

	// First collect all files to be released.
	var archiveFilenames []string

	for _, archPath := range release.ArchsCompiled {
		archiveDir := filepath.Join(
			b.core.DistDir,
			b.core.Config.Project,
			b.core.Tag,
			b.core.DistRootArchives,
			filepath.FromSlash(archPath.Path),
		)
		archiveFilenames = append(archiveFilenames, filepath.Join(archiveDir, archPath.Name))

	}

	if len(archiveFilenames) == 0 {
		return fmt.Errorf("%s: no files found for release %q", commandName, release.Path)
	}

	if b.core.Try {
		return nil
	}

	checksumFilename, err := b.generateChecksumTxt(rctx, archiveFilenames...)
	if err != nil {
		return err
	}

	archiveFilenames = append(archiveFilenames, checksumFilename)

	logCtx.Logf("Prepared %d files to archive: %v", len(archiveFilenames), archiveFilenames)

	// Generate release notes if needed.
	// Write them to the release dir in dist to make testing easier.
	if info.Settings.ReleaseNotesSettings.Generate {
		releaseNotesFilename, err := b.generateReleaseNotes(rctx)
		if err != nil {
			return err
		}
		if releaseNotesFilename == "" {
			panic("releaseNotesFilename is empty")
		}
		info.Settings.ReleaseNotesSettings.Filename = releaseNotesFilename
	}

	// Now create the release archive and upload files.
	releaseID, err := client.Release(ctx, info)
	if err != nil {
		return fmt.Errorf("%s: failed to create release: %v", commandName, err)
	}
	r, ctx := b.core.Workforce.Start(ctx)

	for _, archiveFilename := range archiveFilenames {
		archiveFilename := archiveFilename
		r.Run(func() error {
			openFile := func() (*os.File, error) {
				return os.Open(archiveFilename)
			}
			logCtx.Logf("Uploading release file %s", archiveFilename)
			if err := releases.UploadAssetsFileWithRetries(ctx, client, info, releaseID, openFile); err != nil {
				return err
			}
			return nil
		})
	}

	if err := r.Wait(); err != nil {
		return fmt.Errorf("%s: failed to upload files: %v", commandName, err)
	}

	return nil
}

func (b *Releaser) generateReleaseNotes(rctx releaseContext) (string, error) {
	if rctx.Info.Settings.ReleaseNotesSettings.Filename != "" {
		return "", fmt.Errorf("%s: both GenerateReleaseNotes and ReleaseNotesFilename are set for release type %q", commandName, rctx.Info.Settings.Type)
	}

	var resolveUsername func(commit, author string) (string, error)
	if unc, ok := rctx.Client.(releases.UsernameResolver); ok {
		resolveUsername = func(commit, author string) (string, error) {
			return unc.ResolveUsername(rctx.Ctx, commit, author, rctx.Info)
		}
	}

	infos, err := changelog.CollectChanges(
		changelog.Options{
			Tag:             b.core.Tag,
			Commitish:       b.commitish,
			RepoPath:        os.Getenv("HUGORELEASER_CHANGELOG_GITREPO"), // Set in tests.
			ResolveUserName: resolveUsername,
		},
	)
	if err != nil {
		return "", err
	}

	changeGroups := rctx.Info.Settings.ReleaseNotesSettings.Groups
	shortThreshold := rctx.Info.Settings.ReleaseNotesSettings.ShortThreshold
	if shortThreshold > 0 && len(infos) < shortThreshold {
		shortTitle := rctx.Info.Settings.ReleaseNotesSettings.ShortTitle
		if shortTitle == "" {
			shortTitle = "What's Changed"
		}
		changeGroups = []config.ReleaseNotesGroup{
			{
				Title:          shortTitle,
				RegexpCompiled: matchers.MatchEverything,
			},
		}
	}

	infosGrouped, err := changelog.GroupByTitleFunc(infos, func(change changelog.Change) (string, int, bool) {
		for i, g := range changeGroups {
			if g.RegexpCompiled.Match(change.Subject) {
				if g.Ignore {
					return "", 0, false
				}
				ordinal := g.Ordinal
				if ordinal == 0 {
					ordinal = i + 1
				}
				return g.Title, ordinal, true
			}
		}
		return "", 0, false
	})

	if err != nil {
		return "", err
	}

	type ReleaseNotesContext struct {
		ChangeGroups []changelog.TitleChanges
	}

	rnc := ReleaseNotesContext{
		ChangeGroups: infosGrouped,
	}

	releaseNotesFilename := filepath.Join(rctx.ReleaseDir, "release-notes.md")
	rctx.Info.Settings.ReleaseNotesSettings.Filename = releaseNotesFilename
	err = func() error {
		f, err := os.Create(releaseNotesFilename)
		if err != nil {
			return err
		}
		defer f.Close()

		var t *template.Template

		if customTemplateFilename := rctx.Info.Settings.ReleaseNotesSettings.TemplateFilename; customTemplateFilename != "" {
			if !filepath.IsAbs(customTemplateFilename) {
				customTemplateFilename = filepath.Join(b.core.ProjectDir, customTemplateFilename)
			}
			b, err := os.ReadFile(customTemplateFilename)
			if err != nil {
				return err
			}
			t, err = templ.Parse(string(b))
			if err != nil {
				return err
			}
		} else {
			t = staticfiles.ReleaseNotesTemplate

		}

		if err := t.Execute(f, rnc); err != nil {
			return err
		}

		return nil
	}()

	if err != nil {
		return "", fmt.Errorf("%s: failed to create release notes file %q: %s", commandName, releaseNotesFilename, err)
	}

	rctx.Log.WithField("filename", releaseNotesFilename).Log(logg.String("Created release notes"))

	return releaseNotesFilename, nil
}

func (b *Releaser) generateChecksumTxt(rctx releaseContext, archiveFilenames ...string) (string, error) {
	// Create a checksums.txt file.
	checksumLines, err := releases.CreateChecksumLines(b.core.Workforce, archiveFilenames...)
	if err != nil {
		return "", err
	}
	// This is what Hugo got out of the box from Goreleaser. No settings for now.
	name := fmt.Sprintf("%s_%s_checksums.txt", rctx.Info.Project, strings.TrimPrefix(rctx.Info.Tag, "v"))

	checksumFilename := filepath.Join(rctx.ReleaseDir, name)
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
		return "", fmt.Errorf("%s: failed to create checksum file %q: %s", commandName, checksumFilename, err)
	}

	rctx.Log.WithField("filename", checksumFilename).Log(logg.String("Created checksum file"))

	return checksumFilename, nil
}
