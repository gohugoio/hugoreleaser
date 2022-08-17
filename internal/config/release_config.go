package config

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/bep/hugoreleaser/internal/common/matchers"
	"github.com/bep/hugoreleaser/internal/releases/releasetypes"
	"github.com/gobwas/glob"
)

type Release struct {
	Paths string `toml:"paths"`

	// Dir is the directory below /dist/releases where the release artifacts gets stored.
	// This must be unique for each release within one configuration file.
	Dir string `toml:"dir"`

	ReleaseSettings ReleaseSettings `toml:"release_settings"`

	PathsCompiled matchers.Matcher `toml:"-"`
}

func (a *Release) Init() error {
	what := "releases"

	if a.Dir == "" {
		return fmt.Errorf("%s: dir is required", what)
	}

	a.Dir = path.Clean(filepath.ToSlash(a.Dir))

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

	// Meta is release type specific metadata.
	Meta map[string]any `toml:"meta"`

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
