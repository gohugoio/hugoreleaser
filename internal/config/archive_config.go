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
	"strings"

	"github.com/gohugoio/hugoreleaser/internal/archives/archiveformats"
	"github.com/gohugoio/hugoreleaser/internal/common/matchers"
	"github.com/gohugoio/hugoreleaser/plugins/model"
)

var (
	_ model.Initializer = (*Archive)(nil)
	_ model.Initializer = (*ArchiveSettings)(nil)
	_ model.Initializer = (*ArchiveType)(nil)
)

type Archive struct {
	// Glob of Build paths to archive. Multiple paths will be ANDed.
	Paths           []string        `toml:"paths"`
	ArchiveSettings ArchiveSettings `toml:"archive_settings"`

	PathsCompiled matchers.Matcher `toml:"-"`
	ArchsCompiled []BuildArchPath  `toml:"-"`
}

func (a *Archive) Init() error {
	what := fmt.Sprintf("archives: %v", a.Paths)

	const prefix = "builds/"
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

	if err := a.ArchiveSettings.Init(); err != nil {
		return fmt.Errorf("%s: %v", what, err)
	}

	return nil
}

type BuildArchPath struct {
	Arch BuildArch `toml:"arch"`
	Path string    `toml:"path"`

	// Name is the name of the archive with the extension.
	Name string `toml:"name"`
}

type ArchiveSettings struct {
	Type ArchiveType `toml:"type"`

	BinaryDir    string            `toml:"binary_dir"`
	NameTemplate string            `toml:"name_template"`
	ExtraFiles   []ArchiveFileInfo `toml:"extra_files"`
	Replacements map[string]string `toml:"replacements"`
	Plugin       Plugin            `toml:"plugin"`

	// CustomSettings is archive type specific metadata.
	// See in the documentation for the configured archive type.
	CustomSettings map[string]any `toml:"custom_settings"`

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
	if a.FormatParsed, err = archiveformats.Parse(a.Format); err != nil {
		return err
	}

	return nil
}

// IsZero is needed to get the shallow merge correct.
func (a ArchiveType) IsZero() bool {
	return a.Format == "" && a.Extension == ""
}

type Archives []Archive
