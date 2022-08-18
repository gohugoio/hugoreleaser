package config

import (
	"fmt"
	"path"

	"github.com/bep/hugoreleaser/internal/builds"
	"github.com/bep/hugoreleaser/pkg/model"
	"github.com/bep/logg"
)

var (
	_ model.Initializer = (*Build)(nil)
)

type Build struct {
	Path string    `toml:"path"`
	Os   []BuildOs `toml:"os"`

	BuildSettings BuildSettings `toml:"build_settings"`
}

func (b *Build) Init() error {
	for _, os := range b.Os {
		for _, arch := range os.Archs {
			if arch.Goarch == builds.UniversalGoarch && os.Goos != "darwin" {
				return fmt.Errorf("universal arch is only supported on MacOS (GOOS=darwin)")
			}
		}
	}
	return nil
}

func (b Build) IsZero() bool {
	return b.Path == "" && len(b.Os) == 0
}

var _ logg.Fielder = BuildSettings{}

type BuildSettings struct {
	Binary string `toml:"binary"`

	Env     []string `toml:"env"`
	Ldflags string   `toml:"ldflags"`
	Flags   []string `toml:"flags"`

	Goarm string `toml:"goarm"`

	GoExe   string `toml:"go_exe"`
	GoProxy string `toml:"go_proxy"`
}

func (b BuildSettings) Fields() logg.Fields {
	return logg.Fields{
		logg.Field{Name: "flags", Value: b.Flags},
		logg.Field{Name: "ldflags", Value: b.Ldflags},
	}
}

type Builds []Build

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
