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
	"os"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestDecode(t *testing.T) {
	c := qt.New(t)

	c.Run("Invalid archive format", func(c *qt.C) {
		file := `
[[archives]]
[archives.archive_settings]
format = "foo"
`

		_, err := DecodeAndApplyDefaults(strings.NewReader(file))
		c.Assert(err, qt.Not(qt.IsNil))
	})
}

func TestDecodeFile(t *testing.T) {
	c := qt.New(t)

	f, err := os.Open("../../hugoreleaser.yaml")
	c.Assert(err, qt.IsNil)
	defer f.Close()

	cfg, err := DecodeAndApplyDefaults(f)
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	c.Assert(err, qt.IsNil)
	c.Assert(cfg.Project, qt.Equals, "hugoreleaser")

	assertHasBuildSettings := func(b BuildSettings) {
		c.Helper()
		c.Assert(b.Env, qt.IsNotNil)
		c.Assert(b.Flags, qt.IsNotNil)
		c.Assert(b.GoSettings.GoProxy, qt.Not(qt.Equals), "")
		c.Assert(b.GoSettings.GoExe, qt.Not(qt.Equals), "")
	}

	assertHasBuildSettings(cfg.BuildSettings)
	for _, b := range cfg.Builds {
		assertHasBuildSettings(b.BuildSettings)
		for _, o := range b.Os {
			assertHasBuildSettings(o.BuildSettings)
			for _, a := range o.Archs {
				assertHasBuildSettings(a.BuildSettings)
			}
		}
	}
}
