package config

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/bep/hugoreleaser/internal/archives/archiveformats"
	"github.com/bep/hugoreleaser/internal/common/matchers"
	"github.com/bep/hugoreleaser/pkg/model"
	"github.com/gobwas/glob"
)

var (
	_ model.Initializer = (*Archive)(nil)
	_ model.Initializer = (*ArchiveSettings)(nil)
	_ model.Initializer = (*ArchiveType)(nil)
)

type Archive struct {
	// Glob of Build paths to archive.
	Paths           string          `toml:"paths"`
	ArchiveSettings ArchiveSettings `toml:"archive_settings"`

	PathsCompiled matchers.Matcher `toml:"-"`
}

func (a *Archive) Init() error {
	what := path.Join("archives", a.Paths)

	const prefix = "/builds/"

	if !strings.HasPrefix(a.Paths, prefix) {
		return fmt.Errorf("%s: archive paths must start with %s", what, prefix)
	}

	// Strip the /builds/ prefix. We currently don't use that,
	// it's just there to make the config easier to understand.
	paths := strings.TrimPrefix(a.Paths, prefix)

	if paths == "" {
		return fmt.Errorf("archive has no paths")
	}

	var err error
	a.PathsCompiled, err = glob.Compile(paths)
	if err != nil {
		return fmt.Errorf("failed to compile archive paths glob %q: %v", a.Paths, err)
	}

	if err := a.ArchiveSettings.Init(); err != nil {
		return fmt.Errorf("%s: %v", what, err)
	}

	return nil
}

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
	case archiveformats.Plugin:
		if err := a.Plugin.Init(); err != nil {
			return fmt.Errorf("%s: %v", what, err)
		}
	default:
		// Clear it to we don't need to start it.
		a.Plugin.Clear()

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

	FormatParsed        archiveformats.Format `toml:"-"`
	HeaderModTimeParsed time.Time             `toml:"-"`
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
