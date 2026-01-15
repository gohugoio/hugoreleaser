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
	"github.com/gohugoio/hugoreleaser/cmd/corecmd"
	"github.com/gohugoio/hugoreleaser/internal/common/templ"
	"github.com/gohugoio/hugoreleaser/internal/config"
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
		ShortHelp:  "Publish a draft release and update package managers.",
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

	// Get release settings from config.
	if len(p.core.Config.Releases) == 0 {
		return fmt.Errorf("%s: no releases defined in config", commandName)
	}

	// Use first release for settings (consistent with release command behavior).
	release := p.core.Config.Releases[0]
	settings := release.ReleaseSettings

	logFields := logg.Fields{
		{Name: "tag", Value: p.core.Tag},
		{Name: "repository", Value: fmt.Sprintf("%s/%s", settings.RepositoryOwner, settings.Repository)},
	}
	logCtx := p.infoLog.WithFields(logFields)

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

	// Step 1: Check and publish the GitHub release.
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

	// Step 2: Update Homebrew cask if enabled.
	caskSettings := p.core.Config.PublishSettings.HomebrewCask
	if caskSettings.Enabled {
		if err := p.updateHomebrewCask(ctx, logCtx, client, release, caskSettings); err != nil {
			return fmt.Errorf("%s: failed to update Homebrew cask: %v", commandName, err)
		}
	}

	return nil
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
	release config.Release,
	caskSettings config.HomebrewCaskSettings,
) error {
	logCtx = logCtx.WithField("action", "homebrew-cask")
	logCtx.Log(logg.String("Updating Homebrew cask"))

	releaseSettings := release.ReleaseSettings
	version := strings.TrimPrefix(p.core.Tag, "v")

	// Find the first .pkg archive matching the path pattern.
	pkgInfo, err := p.findPkgArchive(release, caskSettings)
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
		Name:             caskSettings.Name,
		DisplayName:      p.core.Config.Project,
		Version:          version,
		SHA256:           pkgInfo.SHA256,
		URL:              downloadURL,
		Description:      caskSettings.Description,
		Homepage:         caskSettings.Homepage,
		PkgFilename:      pkgInfo.Name,
		BundleIdentifier: caskSettings.BundleIdentifier,
	}

	// Render cask template.
	var caskContent bytes.Buffer
	var tmpl *template.Template

	if caskSettings.TemplateFilename != "" {
		templatePath := caskSettings.TemplateFilename
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
	commitMessage := fmt.Sprintf("Update %s to %s", caskSettings.Name, p.core.Tag)

	logCtx.WithFields(logg.Fields{
		{Name: "tap", Value: fmt.Sprintf("%s/%s", releaseSettings.RepositoryOwner, caskSettings.TapRepository)},
		{Name: "path", Value: caskSettings.CaskPath},
	}).Log(logg.String("Committing cask update"))

	if p.core.Try {
		logCtx.Log(logg.String("Trial run - skipping commit"))
		return nil
	}

	sha, err := client.UpdateFileInRepo(
		ctx,
		releaseSettings.RepositoryOwner,
		caskSettings.TapRepository,
		caskSettings.CaskPath,
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

// findPkgArchive finds the first .pkg archive for darwin matching the path pattern.
func (p *Publisher) findPkgArchive(release config.Release, caskSettings config.HomebrewCaskSettings) (pkgArchiveInfo, error) {
	pathMatcher := caskSettings.PathCompiled

	for _, archPath := range release.ArchsCompiled {
		// Only consider darwin archives.
		if archPath.Arch.Os == nil || archPath.Arch.Os.Goos != "darwin" {
			continue
		}

		// Check if the path matches the pattern.
		if pathMatcher != nil && !pathMatcher.Match(archPath.Path) {
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

	return pkgArchiveInfo{}, fmt.Errorf("no .pkg archive found for darwin matching path pattern %q", caskSettings.Path)
}
