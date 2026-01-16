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

package config

import (
	"fmt"
	"strings"

	"github.com/gohugoio/hugoreleaser/internal/common/matchers"
	"github.com/gohugoio/hugoreleaser/internal/publish/publishformats"
)

// PublishSettings contains shared defaults for publishers.
type PublishSettings struct {
	// Shared settings that can be overridden per publisher (extensible).
}

func (p *PublishSettings) Init() error {
	return nil
}

// Publisher represents a single publish target.
type Publisher struct {
	// Paths with glob patterns to match releases and optionally filter archives.
	// Format: releases/<release-pattern>[/archives/<archive-pattern>]
	// Examples:
	// - "releases/**" matches all releases, all archives
	// - "releases/myrelease/**" matches specific release, all archives
	// - "releases/**/archives/macos/**" matches all releases, only macos archives
	// - "releases/myrelease/archives/macos/**" matches specific release, only macos archives
	// Default (no paths): matches all releases and all archives.
	Paths []string `json:"paths"`

	Type   PublishType `json:"type"`
	Plugin Plugin      `json:"plugin"`

	// CustomSettings contains type-specific settings.
	// For homebrew_cask: bundle_identifier, tap_repository, name, cask_path, etc.
	CustomSettings map[string]any `json:"custom_settings"`

	// Compiled fields
	ReleasePathsCompiled matchers.Matcher `json:"-"`
	ArchivePathsCompiled matchers.Matcher `json:"-"`
	ReleasesCompiled     []*Release       `json:"-"`
}

func (p *Publisher) Init() error {
	what := fmt.Sprintf("publishers: %v", p.Paths)

	if err := p.Type.Init(); err != nil {
		return fmt.Errorf("%s: %v", what, err)
	}

	// Validate format setup.
	switch p.Type.FormatParsed {
	case publishformats.Plugin:
		if err := p.Plugin.Init(); err != nil {
			return fmt.Errorf("%s: %v", what, err)
		}
	default:
		// Clear it so we don't need to start it.
		p.Plugin.Clear()
	}

	// Parse unified path format: releases/<release-pattern>[/archives/<archive-pattern>]
	var releasePaths, archivePaths []string
	const (
		releasesPrefix    = "releases/"
		archivesSeparator = "/archives/"
	)

	for _, path := range p.Paths {
		if !strings.HasPrefix(path, releasesPrefix) {
			return fmt.Errorf("%s: paths must start with %q, got %q", what, releasesPrefix, path)
		}

		rest := path[len(releasesPrefix):]

		// Check if path contains "/archives/" separator
		if idx := strings.Index(rest, archivesSeparator); idx != -1 {
			releasePart := rest[:idx]
			archivePart := rest[idx+len(archivesSeparator):]
			if releasePart == "" {
				releasePart = "**"
			}
			releasePaths = append(releasePaths, releasePart)
			archivePaths = append(archivePaths, archivePart)
		} else {
			// No archive filter - match all archives
			releasePaths = append(releasePaths, rest)
		}
	}

	// Compile release paths if provided.
	if len(releasePaths) > 0 {
		var err error
		p.ReleasePathsCompiled, err = matchers.Glob(releasePaths...)
		if err != nil {
			return fmt.Errorf("%s: failed to compile release paths glob: %v", what, err)
		}
	}

	// Compile archive paths if provided.
	if len(archivePaths) > 0 {
		var err error
		p.ArchivePathsCompiled, err = matchers.Glob(archivePaths...)
		if err != nil {
			return fmt.Errorf("%s: failed to compile archive paths glob: %v", what, err)
		}
	}

	// Validate type-specific settings.
	switch p.Type.FormatParsed {
	case publishformats.HomebrewCask:
		if err := p.validateHomebrewCaskSettings(); err != nil {
			return fmt.Errorf("%s: %v", what, err)
		}
	}

	return nil
}

func (p *Publisher) validateHomebrewCaskSettings() error {
	what := "homebrew_cask"

	// bundle_identifier is required.
	if _, ok := p.CustomSettings["bundle_identifier"]; !ok {
		return fmt.Errorf("%s: bundle_identifier is required in custom_settings", what)
	}

	return nil
}

// PublishType represents the type of publisher.
type PublishType struct {
	Format string `json:"format"` // github_release, homebrew_cask, _plugin

	FormatParsed publishformats.Format `json:"-"`
}

func (t *PublishType) Init() error {
	what := "type"
	if t.Format == "" {
		return fmt.Errorf("%s: has no format", what)
	}

	var err error
	if t.FormatParsed, err = publishformats.Parse(t.Format); err != nil {
		return err
	}

	return nil
}

// IsZero is needed to get the shallow merge correct.
func (t PublishType) IsZero() bool {
	return t.Format == ""
}

// Publishers is a slice of Publisher.
type Publishers []Publisher
