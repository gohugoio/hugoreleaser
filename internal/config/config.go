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
	"io/fs"

	"github.com/gohugoio/hugoreleaser/internal/common/matchers"
	"github.com/gohugoio/hugoreleaser/internal/plugins/plugintypes"
)

type Config struct {
	// A bucket for anchors that defines reusable YAML fragments.
	Definitions map[string]any ` json:"definitions"`

	Project                  string            `json:"project"`
	ArchiveAliasReplacements map[string]string `json:"archive_alias_replacements"`

	GoSettings GoSettings `json:"go_settings"`

	Builds   Builds   `json:"builds"`
	Archives Archives `json:"archives"`
	Releases Releases `json:"releases"`

	BuildSettings   BuildSettings   `json:"build_settings"`
	ArchiveSettings ArchiveSettings `json:"archive_settings"`
	ReleaseSettings ReleaseSettings `json:"release_settings"`
}

func (c Config) FindReleases(filter matchers.Matcher) []Release {
	var releases []Release
	for _, release := range c.Releases {
		if filter == nil || filter.Match(release.Path) {
			releases = append(releases, release)
		}
	}
	return releases
}

// FindArchs returns the archs that match the given filter
func (c Config) FindArchs(filter matchers.Matcher) []BuildArchPath {
	var archs []BuildArchPath
	for _, build := range c.Builds {
		buildPath := build.Path
		for _, os := range build.Os {
			osPath := buildPath + "/" + os.Goos
			for _, arch := range os.Archs {
				archPath := osPath + "/" + arch.Goarch
				if filter.Match(archPath) {
					archs = append(archs, BuildArchPath{Arch: arch, Path: archPath})
				}
			}
		}
	}
	return archs
}

type Plugin struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"`
	Command string   `json:"command"`
	Dir     string   `json:"dir"`
	Env     []string `json:"env"`

	TypeParsed plugintypes.Type `json:"-"`
}

func (t *Plugin) Clear() {
	t.ID = ""
	t.Type = ""
	t.Command = ""
	t.Dir = ""
	t.TypeParsed = plugintypes.Invalid
}

func (t *Plugin) Init() error {
	what := "plugin"
	if t.ID == "" {
		return fmt.Errorf("%s: has no id", what)
	}
	if t.Command == "" {
		return fmt.Errorf("%s: %q has no command", what, t.ID)
	}

	var err error
	if t.TypeParsed, err = plugintypes.Parse(t.Type); err != nil {
		return fmt.Errorf("%s: %v", what, err)
	}

	return nil
}

func (t Plugin) IsZero() bool {
	return t.ID == ""
}

type ArchiveFileInfo struct {
	SourcePath string      `json:"source_path"`
	TargetPath string      `json:"target_path"`
	Mode       fs.FileMode `json:"mode"`
}
