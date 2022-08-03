package config

import (
	"fmt"
	"path"
	"strings"

	"github.com/bep/hugoreleaser/internal/archives/archiveformats"
	"github.com/gobwas/glob"
)

type Archive struct {
	Paths           string          `toml:"paths"`
	ArchiveSettings ArchiveSettings `toml:"archive_settings"`

	PathsCompiled glob.Glob `toml:"-"`
}

func (a *Archive) init() error {
	if a.Paths == "" {
		return fmt.Errorf("archive has no paths")
	}

	var err error
	a.PathsCompiled, err = glob.Compile(a.Paths)
	if err != nil {
		return fmt.Errorf("failed to compile archive paths glob %q: %v", a.Paths, err)
	}

	return a.ArchiveSettings.init()
}

// ArchiveMeta is used by the Deb archive format.
type ArchiveMeta struct {
	Vendor      string `toml:"vendor"`
	Homepage    string `toml:"homepage"`
	Maintainer  string `toml:"maintainer"`
	Description string `toml:"description"`
	License     string `toml:"license"`
}

type ArchiveSettings struct {
	Format       string            `toml:"format"`
	NameTemplate string            `toml:"name_template"`
	ExtraFiles   []string          `toml:"extra_files"`
	Replacements map[string]string `toml:"replacements"`
	Meta         ArchiveMeta       `toml:"meta"`
	ExternalTool ExternalTool      `toml:"external_tool"`

	ReplacementsReplacer *strings.Replacer     `toml:"-"`
	Formati              archiveformats.Format `toml:"-"`
}

type ExternalTool struct {
	Name    string `toml:"name"`
	Type    string `toml:"type"`
	Command string `toml:"command"`
}

func (t *ExternalTool) init() error {
	op := "external_tool"
	if t.Name == "" {
		return fmt.Errorf("%s: has no name", op)
	}
	if t.Type == "" {
		return fmt.Errorf("%s: %q has no type", op, t.Name)
	}
	if t.Command == "" {
		return fmt.Errorf("%s: %q has no command", op, t.Name)
	}
	return nil
}

func (a *ArchiveSettings) init() error {
	op := "archive_settings"

	var err error
	if a.Formati, err = archiveformats.ParseFormat(a.Format); err != nil {
		return err
	}

	// Validate format setup.
	switch a.Formati {
	case archiveformats.External:
		if err := a.ExternalTool.init(); err != nil {
			return fmt.Errorf("%s: %v", op, err)
		}
	}

	var oldNew []string
	for k, v := range a.Replacements {
		oldNew = append(oldNew, k, v)
	}

	a.ReplacementsReplacer = strings.NewReplacer(oldNew...)

	return nil
}

type Archives []Archive

type Build struct {
	Path string    `toml:"path"`
	Os   []BuildOs `toml:"os"`

	BuildSettings BuildSettings `toml:"build_settings"`
}

func (b Build) IsZero() bool {
	return b.Path == "" && len(b.Os) == 0
}

type BuildArch struct {
	Goarch string `toml:"goarch"`

	BuildSettings BuildSettings `toml:"build_settings"`

	// Tree navigation.
	Build *Build   `toml:"-"`
	Os    *BuildOs `toml:"-"`
}

// BinaryPath returns the path to the built binary starting below /builds.
func (b BuildArch) BinaryPath() string {
	return path.Join(b.Build.Path, b.Os.Goos, b.Goarch, b.BuildSettings.Binary)
}

type BuildOs struct {
	Goos  string      `toml:"goos"`
	Archs []BuildArch `toml:"archs"`

	BuildSettings BuildSettings `toml:"build_settings"`

	// Tree navigation.
	Build *Build `toml:"-"`
}

type BuildSettings struct {
	Binary string `toml:"binary"`

	Env     []string `toml:"env"`
	Ldflags string   `toml:"ldflags"`
	Flags   []string `toml:"flags"`

	Goarm string `toml:"goarm"`

	GoExe   string `toml:"go_exe"`
	GoProxy string `toml:"go_proxy"`
}

type Builds []Build

type Config struct {
	// The configuration version.
	Version string `toml:"version"`

	Project  string   `toml:"project"`
	Builds   Builds   `toml:"builds"`
	Archives Archives `toml:"archives"`

	BuildSettings   BuildSettings   `toml:"build_settings"`
	ArchiveSettings ArchiveSettings `toml:"archive_settings"`
}

// GlobArchs returns the archs that match the given path pattern.
// Note that the build paths starts at '/builds'.
func (c Config) GlobArchs(pattern glob.Glob) []BuildArch {
	const rootPath = "/builds/"

	var archs []BuildArch
	for _, build := range c.Builds {
		buildPath := rootPath + build.Path

		for _, os := range build.Os {
			osPath := buildPath + "/" + os.Goos
			for _, arch := range os.Archs {
				archPath := osPath + "/" + arch.Goarch
				if pattern.Match(archPath) {
					archs = append(archs, arch)
				}
			}
		}
	}
	return archs
}
