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

package archiveformats

import (
	"fmt"
	"strings"

	"github.com/gohugoio/hugoreleaser/internal/common/mapsh"
)

// Goreleaser supports `tar.gz`, `tar.xz`, `tar`, `gz`, `zip` and `binary`.
// We currently limit ourselves to what Hugo uses: `tar.gz` and 'zip' (for Windows).
const (
	InvalidFormat Format = iota
	Deb
	TarGz
	Zip
	Rename
	Plugin // Plugin is a special format that is used to indicate that the archive operation is handled by an external tool.
)

var formatString = map[Format]string{
	// The string values is what users can specify in the config.
	Deb:    "deb",
	TarGz:  "tar.gz",
	Zip:    "zip",
	Rename: "rename",
	Plugin: "_plugin",
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
		return f, fmt.Errorf("invalid archive format %q, must be one of %s", s, mapsh.KeysSorted(formatString))
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

// Format represents the type of archive.
type Format int

func (f Format) String() string {
	return formatString[f]
}
