package corecmd

import (
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/gohugoio/hugoreleaser/internal/common/matchers"
)

func TestCompilePaths(t *testing.T) {
	c := qt.New(t)

	checkMatchesEverything := func(core *Core) bool {
		for _, m := range []matchers.Matcher{core.PathsBuildsCompiled, core.PathsReleasesCompiled} {
			if !m.Match("aasdfasd32dfasdfasdf") {
				return false
			}
		}
		return true
	}

	for _, test := range []struct {
		name   string
		paths  []string
		assert func(c *qt.C, core *Core)
	}{
		{
			name:  "empty",
			paths: []string{},
			assert: func(c *qt.C, core *Core) {
				c.Assert(checkMatchesEverything(core), qt.IsTrue)
			},
		},
		{
			name:  "double asterisk",
			paths: []string{"**"},
			assert: func(c *qt.C, core *Core) {
				c.Assert(checkMatchesEverything(core), qt.IsTrue)
			},
		},
		{
			name:  "match some builds",
			paths: []string{"builds/foo*"},
			assert: func(c *qt.C, core *Core) {
				c.Assert(core.PathsBuildsCompiled.Match("foos"), qt.IsTrue)
				c.Assert(core.PathsBuildsCompiled.Match("bar"), qt.IsFalse)
				c.Assert(core.PathsReleasesCompiled.Match("adfasdfasfd"), qt.IsTrue)

			},
		},
		{
			name:  "match some releases",
			paths: []string{"releases/foo*"},
			assert: func(c *qt.C, core *Core) {
				c.Assert(core.PathsReleasesCompiled.Match("foo"), qt.IsTrue)
				c.Assert(core.PathsReleasesCompiled.Match("bar"), qt.IsFalse)
				c.Assert(core.PathsBuildsCompiled.Match("adfasdfasfd"), qt.IsTrue)

			},
		},
		{
			name:  "match some builds and releases",
			paths: []string{"releases/foo*", "builds/bar*"},
			assert: func(c *qt.C, core *Core) {
				c.Assert(core.PathsReleasesCompiled.Match("foos"), qt.IsTrue)
				c.Assert(core.PathsReleasesCompiled.Match("bars"), qt.IsFalse)
				c.Assert(core.PathsReleasesCompiled.Match("asdfasdf"), qt.IsFalse)
				c.Assert(core.PathsBuildsCompiled.Match("adfasdfasfd"), qt.IsFalse)

			},
		},
		{
			name:  "multiple release paths",
			paths: []string{"releases/foo*", "releases/**.zip"},
			assert: func(c *qt.C, core *Core) {
				c.Assert(core.PathsReleasesCompiled.Match("foos.zip"), qt.IsTrue)

			},
		},
	} {
		c.Run(test.name, func(c *qt.C) {
			core := &Core{
				Paths: test.paths,
			}
			c.Assert(core.compilePaths(), qt.IsNil)
			test.assert(c, core)
		})
	}

	c.Assert((&Core{Paths: []string{"/**"}}).compilePaths(), qt.Not(qt.IsNil))

}
