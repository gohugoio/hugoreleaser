// Copyright 2026 The Hugoreleaser Authors
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

package publishformats

import (
	"fmt"
	"strings"

	"github.com/gohugoio/hugoreleaser/internal/common/mapsh"
)

const (
	InvalidFormat Format = iota
	GitHubRelease        // Undrafts the GitHub release
	HomebrewCask         // Updates Homebrew cask file
	Plugin               // Plugin is a special format handled by an external tool
)

var formatString = map[Format]string{
	// The string values is what users can specify in the config.
	GitHubRelease: "github_release",
	HomebrewCask:  "homebrew_cask",
	Plugin:        "_plugin",
}

var stringFormat = map[string]Format{}

func init() {
	for k, v := range formatString {
		stringFormat[v] = k
	}
}

// Parse parses a string into a Format.
func Parse(s string) (Format, error) {
	f := stringFormat[strings.ToLower(s)]
	if f == InvalidFormat {
		return f, fmt.Errorf("invalid publish format %q, must be one of %s", s, mapsh.KeysSorted(formatString))
	}
	return f, nil
}

// MustParse parses a string into a Format and panics if it fails.
func MustParse(s string) Format {
	f, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return f
}

// Format represents the type of publisher.
type Format int

func (f Format) String() string {
	return formatString[f]
}
