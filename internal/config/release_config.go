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
	"regexp"
	"strings"

	"github.com/gohugoio/hugoreleaser/internal/common/matchers"
	"github.com/gohugoio/hugoreleaser/internal/releases/releasetypes"
)

type Release struct {
	// Paths with Glob of releases paths to release. Multiple paths will be ANDed.
	Paths []string `json:"paths"`

	// Path is the directory below /dist/releases where the release artifacts gets stored.
	// This must be unique for each release within one configuration file.
	Path string `json:"path"`

	ReleaseSettings ReleaseSettings `json:"release_settings"`

	PathsCompiled matchers.Matcher `json:"-"`

	// Builds matching Paths.
	ArchsCompiled []BuildArchPath `json:"-"`
}

func (a *Release) Init() error {
	what := "releases"

	if a.Path == "" {
		return fmt.Errorf("%s: dir is required", what)
	}

	a.Path = path.Clean(filepath.ToSlash(a.Path))

	const prefix = "archives/"
	for i, p := range a.Paths {
		if !strings.HasPrefix(p, prefix) {
			return fmt.Errorf("%s: archive paths must start with %s", what, prefix)
		}
		a.Paths[i] = p[len(prefix):]
	}

	var err error
	a.PathsCompiled, err = matchers.Glob(a.Paths...)
	if err != nil {
		return fmt.Errorf("failed to compile archive paths glob %q: %v", a.Paths, err)
	}

	if err := a.ReleaseSettings.Init(); err != nil {
		return fmt.Errorf("%s: %v", what, err)
	}

	return nil
}

type ReleaseSettings struct {
	Type string `json:"type"`

	Name            string `json:"name"`
	Repository      string `json:"repository"`
	RepositoryOwner string `json:"repository_owner"`
	Draft           bool   `json:"draft"`
	Prerelease      bool   `json:"prerelease"`

	ReleaseNotesSettings ReleaseNotesSettings `json:"release_notes_settings"`

	TypeParsed releasetypes.Type `json:"-"`
}

type ReleaseNotesSettings struct {
	Generate         bool                `json:"generate"`
	GenerateOnHost   bool                `json:"generate_on_host"`
	Filename         string              `json:"filename"`
	TemplateFilename string              `json:"template_filename"`
	Groups           []ReleaseNotesGroup `json:"groups"`

	// Can be used to collapse releases with a few number (less than threshold) of changes into one title.
	ShortThreshold int    `json:"short_threshold"`
	ShortTitle     string `json:"short_title"`
}

func (g *ReleaseNotesSettings) Init() error {
	for i := range g.Groups {
		if err := g.Groups[i].Init(); err != nil {
			return fmt.Errorf("[%d]: %v", i, err)
		}
	}
	return nil
}

type ReleaseNotesGroup struct {
	Title   string `json:"title"`
	Regexp  string `json:"regexp"`
	Ignore  bool   `json:"ignore"`
	Ordinal int    `json:"ordinal"`

	RegexpCompiled matchers.Matcher `json:"-"`
}

func (g *ReleaseNotesGroup) Init() error {
	what := "release.release_settings.group"
	if g.Regexp == "" {
		return fmt.Errorf("%s: regexp is not set", what)
	}

	if !strings.HasPrefix(g.Regexp, "(?") {
		g.Regexp = "(?i)" + g.Regexp
	}

	var err error
	re, err := regexp.Compile(g.Regexp)
	if err != nil {
		return fmt.Errorf("%s: %v", what, err)
	}

	g.RegexpCompiled = matchers.MatcherFunc(func(s string) bool {
		return re.MatchString(s)
	})

	return nil
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

	if len(r.ReleaseNotesSettings.Groups) == 0 {
		// Add a default group matching all.
		r.ReleaseNotesSettings.Groups = []ReleaseNotesGroup{
			{
				Title:  "What's Changed",
				Regexp: ".*",
			},
		}
	}

	if err := r.ReleaseNotesSettings.Init(); err != nil {
		return fmt.Errorf("%s: %v", what, err)
	}

	return nil
}

type Releases []Release
