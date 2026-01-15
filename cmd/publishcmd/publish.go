// Copyright 2026 The Hugoreleaser Authors
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

package publishcmd

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/bep/logg"
	"github.com/gohugoio/hugoreleaser-plugins-api/model"
	"github.com/gohugoio/hugoreleaser/cmd/corecmd"
	"github.com/gohugoio/hugoreleaser/internal/common/matchers"
	"github.com/gohugoio/hugoreleaser/internal/common/templ"
	"github.com/gohugoio/hugoreleaser/internal/config"
	"github.com/gohugoio/hugoreleaser/internal/publish/publishformats"
	"github.com/gohugoio/hugoreleaser/internal/releases"
	"github.com/gohugoio/hugoreleaser/staticfiles"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const commandName = "publish"

// New returns a usable ffcli.Command for the publish subcommand.
func New(core *corecmd.Core) *ffcli.Command {
	fs := flag.NewFlagSet(corecmd.CommandName+" "+commandName, flag.ExitOnError)

	publisher := NewPublisher(core, fs)

	core.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       commandName,
		ShortUsage: corecmd.CommandName + " publish [flags]",
		ShortHelp:  "Publish releases and update package managers.",
		FlagSet:    fs,
		Exec:       publisher.Exec,
	}
}

// NewPublisher returns a new Publisher.
func NewPublisher(core *corecmd.Core, fs *flag.FlagSet) *Publisher {
	return &Publisher{
		core: core,
	}
}

// Publisher handles the publish command.
type Publisher struct {
	core    *corecmd.Core
	infoLog logg.LevelLogger
}

// Init initializes the publisher.
func (p *Publisher) Init() error {
	p.infoLog = p.core.InfoLog.WithField("cmd", commandName)
	return nil
}

// Exec executes the publish command.
func (p *Publisher) Exec(ctx context.Context, args []string) error {
	if err := p.Init(); err != nil {
		return err
	}

	if len(p.core.Config.Publishers) == 0 {
		p.infoLog.Log(logg.String("No publishers configured"))
		return nil
	}

	logFields := logg.Fields{
		{Name: "tag", Value: p.core.Tag},
	}
	logCtx := p.infoLog.WithFields(logFields)

	// Process each publisher.
	for i := range p.core.Config.Publishers {
		pub := &p.core.Config.Publishers[i]

		if len(pub.ReleasesCompiled) == 0 {
			continue
		}

		// Process each release that matches this publisher.
		for _, release := range pub.ReleasesCompiled {
			if err := p.handlePublisher(ctx, logCtx, pub, release); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Publisher) handlePublisher(
	ctx context.Context,
	logCtx logg.LevelLogger,
	pub *config.Publisher,
	release *config.Release,
) error {
	settings := release.ReleaseSettings

	// Create client.
	var client releases.PublishClient
	if p.core.Try {
		client = &releases.FakeClient{}
	} else {
		c, err := releases.NewClient(ctx, settings.TypeParsed)
		if err != nil {
			return fmt.Errorf("%s: failed to create release client: %v", commandName, err)
		}
		var ok bool
		client, ok = c.(releases.PublishClient)
		if !ok {
			return fmt.Errorf("%s: client does not support publish operations", commandName)
		}
	}

	switch pub.Type.FormatParsed {
	case publishformats.GitHubRelease:
		return p.publishGitHubRelease(ctx, logCtx, client, release)
	case publishformats.HomebrewCask:
		return p.updateHomebrewCask(ctx, logCtx, client, pub, release)
	case publishformats.Plugin:
		return fmt.Errorf("%s: plugin publishers not yet implemented", commandName)
	default:
		return fmt.Errorf("%s: unknown publisher format: %s", commandName, pub.Type.Format)
	}
}

func (p *Publisher) publishGitHubRelease(
	ctx context.Context,
	logCtx logg.LevelLogger,
	client releases.PublishClient,
	release *config.Release,
) error {
	settings := release.ReleaseSettings

	logCtx = logCtx.WithFields(logg.Fields{
		{Name: "action", Value: "github_release"},
		{Name: "repository", Value: fmt.Sprintf("%s/%s", settings.RepositoryOwner, settings.Repository)},
	})

	logCtx.Log(logg.String("Checking release status"))

	releaseID, isDraft, err := client.GetReleaseByTag(ctx, settings.RepositoryOwner, settings.Repository, p.core.Tag)
	if err != nil {
		return fmt.Errorf("%s: failed to get release: %v", commandName, err)
	}

	if isDraft {
		logCtx.Log(logg.String("Publishing draft release"))
		if err := client.PublishRelease(ctx, settings.RepositoryOwner, settings.Repository, releaseID); err != nil {
			return fmt.Errorf("%s: failed to publish release: %v", commandName, err)
		}
		logCtx.Log(logg.String("Release published successfully"))
	} else {
		logCtx.Log(logg.String("Release is already published"))
	}

	return nil
}

// HomebrewCaskSettings holds the custom settings for homebrew_cask publisher.
type HomebrewCaskSettings struct {
	BundleIdentifier string `mapstructure:"bundle_identifier"`
	TapRepository    string `mapstructure:"tap_repository"`
	Name             string `mapstructure:"name"`
	CaskPath         string `mapstructure:"cask_path"`
	TemplateFilename string `mapstructure:"template_filename"`
	Description      string `mapstructure:"description"`
	Homepage         string `mapstructure:"homepage"`
}

// HomebrewCaskContext holds data for the Homebrew cask template.
type HomebrewCaskContext struct {
	Name             string
	DisplayName      string
	Version          string
	SHA256           string
	URL              string
	Description      string
	Homepage         string
	PkgFilename      string
	BundleIdentifier string
}

func (p *Publisher) updateHomebrewCask(
	ctx context.Context,
	logCtx logg.LevelLogger,
	client releases.PublishClient,
	pub *config.Publisher,
	release *config.Release,
) error {
	logCtx = logCtx.WithField("action", "homebrew_cask")
	logCtx.Log(logg.String("Updating Homebrew cask"))

	releaseSettings := release.ReleaseSettings
	version := strings.TrimPrefix(p.core.Tag, "v")

	// Read settings from custom_settings.
	settings, err := model.FromMap[any, HomebrewCaskSettings](pub.CustomSettings)
	if err != nil {
		return fmt.Errorf("failed to parse homebrew_cask settings: %w", err)
	}

	// Apply defaults.
	if settings.TapRepository == "" {
		settings.TapRepository = "homebrew-tap"
	}
	if settings.Name == "" {
		settings.Name = p.core.Config.Project
	}
	if settings.CaskPath == "" {
		settings.CaskPath = fmt.Sprintf("Casks/%s.rb", settings.Name)
	}

	// Find the first .pkg archive matching the archive paths pattern.
	pkgInfo, err := p.findPkgArchive(release, pub.ArchivePathsCompiled)
	if err != nil {
		return err
	}

	logCtx.WithField("pkg", pkgInfo.Name).Log(logg.String("Found pkg archive"))

	// Build download URL.
	downloadURL := fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/%s/%s",
		releaseSettings.RepositoryOwner,
		releaseSettings.Repository,
		p.core.Tag,
		pkgInfo.Name,
	)

	// Build cask context.
	caskCtx := HomebrewCaskContext{
		Name:             settings.Name,
		DisplayName:      p.core.Config.Project,
		Version:          version,
		SHA256:           pkgInfo.SHA256,
		URL:              downloadURL,
		Description:      settings.Description,
		Homepage:         settings.Homepage,
		PkgFilename:      pkgInfo.Name,
		BundleIdentifier: settings.BundleIdentifier,
	}

	// Render cask template.
	var caskContent bytes.Buffer
	var tmpl *template.Template

	if settings.TemplateFilename != "" {
		templatePath := settings.TemplateFilename
		if !filepath.IsAbs(templatePath) {
			templatePath = filepath.Join(p.core.ProjectDir, templatePath)
		}
		b, err := os.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("failed to read custom cask template: %v", err)
		}
		tmpl, err = templ.Parse(string(b))
		if err != nil {
			return fmt.Errorf("failed to parse custom cask template: %v", err)
		}
	} else {
		tmpl = staticfiles.HomebrewCaskTemplate
	}

	if err := tmpl.Execute(&caskContent, caskCtx); err != nil {
		return fmt.Errorf("failed to execute cask template: %v", err)
	}

	// Update file in tap repository.
	commitMessage := fmt.Sprintf("Update %s to %s", settings.Name, p.core.Tag)

	logCtx.WithFields(logg.Fields{
		{Name: "tap", Value: fmt.Sprintf("%s/%s", releaseSettings.RepositoryOwner, settings.TapRepository)},
		{Name: "path", Value: settings.CaskPath},
	}).Log(logg.String("Committing cask update"))

	if p.core.Try {
		logCtx.Log(logg.String("Trial run - skipping commit"))
		return nil
	}

	sha, err := client.UpdateFileInRepo(
		ctx,
		releaseSettings.RepositoryOwner,
		settings.TapRepository,
		settings.CaskPath,
		commitMessage,
		caskContent.Bytes(),
	)
	if err != nil {
		return err
	}

	logCtx.WithField("commit", sha).Log(logg.String("Cask updated successfully"))
	return nil
}

// pkgArchiveInfo contains information about a .pkg archive.
type pkgArchiveInfo struct {
	Name   string
	SHA256 string
}

// findPkgArchive finds the first .pkg archive for darwin matching the archive paths pattern.
func (p *Publisher) findPkgArchive(release *config.Release, archivePathsMatcher matchers.Matcher) (pkgArchiveInfo, error) {
	for _, archPath := range release.ArchsCompiled {
		// Only consider darwin archives.
		if archPath.Arch.Os == nil || archPath.Arch.Os.Goos != "darwin" {
			continue
		}

		// Check if the path matches the pattern.
		if archivePathsMatcher != nil && !archivePathsMatcher.Match(archPath.Path) {
			continue
		}

		// Check if it's a .pkg file.
		if strings.HasSuffix(archPath.Name, ".pkg") {
			return pkgArchiveInfo{
				Name:   archPath.Name,
				SHA256: archPath.SHA256,
			}, nil
		}
	}

	return pkgArchiveInfo{}, fmt.Errorf("no .pkg archive found for darwin")
}
