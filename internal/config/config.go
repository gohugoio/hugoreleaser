package config

import (
	"fmt"
	"path"
	"strings"

	"github.com/bep/hugoreleaser/internal/archives/archiveformats"
	"github.com/bep/hugoreleaser/pkg/model"
	"github.com/gobwas/glob"
)

var (
	_ model.Initializer = (*Archive)(nil)
	_ model.Initializer = (*ArchiveSettings)(nil)
	_ model.Initializer = (*ArchiveType)(nil)
)

type Config struct {
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

type Archive struct {
	Paths           string          `toml:"paths"`
	ArchiveSettings ArchiveSettings `toml:"archive_settings"`

	PathsCompiled glob.Glob `toml:"-"`
}

func (a *Archive) Init() error {
	what := path.Join("archives", a.Paths)
	if a.Paths == "" {
		return fmt.Errorf("archive has no paths")
	}

	var err error
	a.PathsCompiled, err = glob.Compile(a.Paths)
	if err != nil {
		return fmt.Errorf("failed to compile archive paths glob %q: %v", a.Paths, err)
	}

	if err := a.ArchiveSettings.Init(); err != nil {
		return fmt.Errorf("%s: %v", what, err)
	}

	return nil
}

// UnmarshalTOML(in any) error

type ArchiveSettings struct {
	Type ArchiveType `toml:"type"`

	BinaryDir    string             `toml:"binary_dir"`
	NameTemplate string             `toml:"name_template"`
	ExtraFiles   []SourceTargetPath `toml:"extra_files"`
	Replacements map[string]string  `toml:"replacements"`
	Plugin       Plugin             `toml:"plugin"`

	// Meta is archive type specific metadata.
	// See in the documentation for the archive type.
	Meta map[string]any `toml:"meta"`

	ReplacementsCompiled *strings.Replacer `toml:"-"`
}

func (a *ArchiveSettings) Init() error {
	what := "archive_settings"

	if err := a.Type.Init(); err != nil {
		return fmt.Errorf("%s: %v", what, err)
	}

	// Validate format setup.
	switch a.Type.FormatParsed {
	case archiveformats.External:
		if err := a.Plugin.Init(); err != nil {
			return fmt.Errorf("%s: %v", what, err)
		}
	}

	var oldNew []string
	for k, v := range a.Replacements {
		oldNew = append(oldNew, k, v)
	}

	a.ReplacementsCompiled = strings.NewReplacer(oldNew...)

	return nil
}

type ArchiveType struct {
	Format    string `toml:"format"`
	Extension string `toml:"extension"`

	FormatParsed archiveformats.Format `toml:"-"`
}

func (a *ArchiveType) Init() error {
	what := "type"
	if a.Format == "" {
		return fmt.Errorf("%s: has no format", what)
	}
	if a.Extension == "" {
		return fmt.Errorf("%s: has no extension", what)
	}
	var err error
	if a.FormatParsed, err = archiveformats.ParseFormat(a.Format); err != nil {
		return err
	}
	return nil
}

// IsZero is needed to get the shallow merge correct.
func (a ArchiveType) IsZero() bool {
	return a.Format == "" && a.Extension == ""
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

type Plugin struct {
	Name    string `toml:"name"`
	Type    string `toml:"type"`
	Command string `toml:"command"`
	Dir     string `toml:"dir"`
}

func (t *Plugin) Init() error {
	what := "plugin"
	if t.Name == "" {
		return fmt.Errorf("%s: has no name", what)
	}
	if t.Type == "" {
		return fmt.Errorf("%s: %q has no type", what, t.Name)
	}
	if t.Command == "" {
		return fmt.Errorf("%s: %q has no command", what, t.Name)
	}
	return nil
}

type SourceTargetPath struct {
	SourcePath string `toml:"source_path"`
	TargetPath string `toml:"target_path"`
}
