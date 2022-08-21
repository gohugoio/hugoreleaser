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

package config

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/gobwas/glob"
	"github.com/gohugoio/hugoreleaser/internal/common/matchers"
	"github.com/gohugoio/hugoreleaser/internal/releases/releasetypes"
)

type Release struct {
	Paths string `toml:"paths"`

	// Path is the directory below /dist/releases where the release artifacts gets stored.
	// This must be unique for each release within one configuration file.
	Path string `toml:"path"`

	ReleaseSettings ReleaseSettings `toml:"release_settings"`

	PathsCompiled matchers.Matcher `toml:"-"`
}

func (a *Release) Init() error {
	what := "releases"

	if a.Path == "" {
		return fmt.Errorf("%s: dir is required", what)
	}

	a.Path = path.Clean(filepath.ToSlash(a.Path))

	const prefix = "/archives/"
	if !strings.HasPrefix(a.Paths, prefix) {
		return fmt.Errorf("%s: release paths must start with %s", what, prefix)
	}
	paths := strings.TrimPrefix(a.Paths, prefix)
	if paths == "" {
		return fmt.Errorf("%s: release has no paths", what)
	}

	var err error
	a.PathsCompiled, err = glob.Compile(paths)
	if err != nil {
		return fmt.Errorf("%s: failed to compile release paths glob %q: %v", what, a.Paths, err)
	}

	if err := a.ReleaseSettings.Init(); err != nil {
		return fmt.Errorf("%s: %v", what, err)
	}

	return nil
}

type ReleaseSettings struct {
	Type string `toml:"type"`

	Name                       string `toml:"name"`
	Repository                 string `toml:"repository"`
	RepositoryOwner            string `toml:"repository_owner"`
	Draft                      bool   `toml:"draft"`
	Prerelease                 bool   `toml:"prerelease"`
	GenerateReleaseNotesOnHost bool   `toml:"generate_release_notes_on_host"`
	ReleaseNotesFilename       string `toml:"release_notes_filename"`

	TypeParsed releasetypes.Type `toml:"-"`
}

func (r *ReleaseSettings) Init() error {
	what := "release.release_settings"
	if r.Type == "" {
		return fmt.Errorf("%s: release type is not set", what)
	}

	var err error
	if r.TypeParsed, err = releasetypes.Parse(r.Type); err != nil {
		return fmt.Errorf("%s: %v", what, err)
	}

	return nil
}

type Releases []Release
