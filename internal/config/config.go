package config

import (
	"fmt"

	"github.com/bep/hugoreleaser/internal/common/matchers"
	"github.com/bep/hugoreleaser/internal/plugins/plugintypes"
	"github.com/gobwas/glob"
)

type Config struct {
	Project  string   `toml:"project"`
	Builds   Builds   `toml:"builds"`
	Archives Archives `toml:"archives"`
	Releases Releases `toml:"releases"`

	BuildSettings   BuildSettings   `toml:"build_settings"`
	ArchiveSettings ArchiveSettings `toml:"archive_settings"`
	ReleaseSettings ReleaseSettings `toml:"release_settings"`
}

func (c Config) FindReleases(filter matchers.Matcher) []Release {
	var releases []Release
	for _, release := range c.Releases {
		if filter == nil || filter.Match(release.Dir) {
			releases = append(releases, release)
		}
	}
	return releases
}

func (c Config) ForEachArchiveArch(inFilter matchers.Matcher, fn func(archive Archive, arch BuildArchPath) error) error {
	for _, archive := range c.Archives {
		filter := archive.PathsCompiled
		if inFilter != nil {
			filter = matchers.And(filter, inFilter)
		}
		archs := c.GlobArchs(filter)
		for _, arch := range archs {
			if err := fn(archive, arch); err != nil {
				return err
			}
		}
	}
	return nil
}

type BuildArchPath struct {
	Arch BuildArch `toml:"arch"`
	Path string    `toml:"path"`
}

// GlobArchs returns the archs that match the given path pattern.
func (c Config) GlobArchs(pattern glob.Glob) []BuildArchPath {
	var archs []BuildArchPath
	for _, build := range c.Builds {
		buildPath := build.Path
		for _, os := range build.Os {
			osPath := buildPath + "/" + os.Goos
			for _, arch := range os.Archs {
				archPath := osPath + "/" + arch.Goarch
				if pattern.Match(archPath) {
					archs = append(archs, BuildArchPath{Arch: arch, Path: archPath})
				}
			}
		}
	}
	return archs
}

type Plugin struct {
	Name    string `toml:"name"`
	Type    string `toml:"type"`
	Command string `toml:"command"`
	Dir     string `toml:"dir"`

	TypeParsed plugintypes.Type `toml:"-"`
}

func (t *Plugin) Clear() {
	t.Name = ""
	t.Type = ""
	t.Command = ""
	t.Dir = ""
	t.TypeParsed = plugintypes.Invalid
}

func (t *Plugin) Init() error {
	what := "plugin"
	if t.Name == "" {
		return fmt.Errorf("%s: has no name", what)
	}
	if t.Command == "" {
		return fmt.Errorf("%s: %q has no command", what, t.Name)
	}

	var err error
	if t.TypeParsed, err = plugintypes.ParseType(t.Type); err != nil {
		return fmt.Errorf("%s: %v", what, err)
	}

	return nil
}

func (t Plugin) IsZero() bool {
	return t.Name == ""
}

type SourceTargetPath struct {
	SourcePath string `toml:"source_path"`
	TargetPath string `toml:"target_path"`
}
