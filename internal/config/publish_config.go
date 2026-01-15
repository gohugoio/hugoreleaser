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
)

// PublishSettings contains settings for the publish command.
type PublishSettings struct {
	HomebrewCask HomebrewCaskSettings `json:"homebrew_cask"`
}

// HomebrewCaskSettings contains settings for updating a Homebrew cask.
type HomebrewCaskSettings struct {
	// Enable or disable Homebrew cask publishing.
	Enabled bool `json:"enabled"`

	// The tap repository name (e.g., "homebrew-tap").
	// Full repo will be: <repository_owner>/<tap_repository>
	TapRepository string `json:"tap_repository"`

	// The cask name (e.g., "hugo").
	Name string `json:"name"`

	// Description shown in the cask.
	Description string `json:"description"`

	// Homepage URL.
	Homepage string `json:"homepage"`

	// The macOS bundle identifier (e.g., "io.gohugo.hugo").
	BundleIdentifier string `json:"bundle_identifier"`

	// Path is a glob pattern to match the archive path for the macOS pkg.
	// This should match a .pkg archive in the release.
	// Example: "macos/**" or "darwin/**"
	// Default: "**" (matches first .pkg found)
	Path string `json:"path"`

	// Path to the cask file in the tap repository.
	// Default: "Casks/<name>.rb"
	CaskPath string `json:"cask_path"`

	// Custom cask template file (optional).
	// If not set, uses built-in template.
	TemplateFilename string `json:"template_filename"`

	// PathCompiled is the compiled matcher for the Path glob.
	PathCompiled matchers.Matcher `json:"-"`
}

func (h *HomebrewCaskSettings) Init(projectName string) error {
	if !h.Enabled {
		return nil
	}

	what := "publish_settings.homebrew_cask"

	// Set defaults.
	if h.TapRepository == "" {
		h.TapRepository = "homebrew-tap"
	}
	if h.Name == "" {
		h.Name = projectName
	}

	if h.BundleIdentifier == "" {
		return fmt.Errorf("%s: bundle_identifier is required", what)
	}

	if h.Path == "" {
		h.Path = "**"
	}
	if h.CaskPath == "" {
		h.CaskPath = fmt.Sprintf("Casks/%s.rb", h.Name)
	}

	// Compile the path matcher.
	// Strip "archives/" prefix if present (similar to release paths).
	path := strings.TrimPrefix(h.Path, "archives/")

	var err error
	h.PathCompiled, err = matchers.Glob(path)
	if err != nil {
		return fmt.Errorf("%s: failed to compile path glob %q: %v", what, h.Path, err)
	}

	return nil
}

func (p *PublishSettings) Init(projectName string) error {
	if err := p.HomebrewCask.Init(projectName); err != nil {
		return err
	}
	return nil
}
