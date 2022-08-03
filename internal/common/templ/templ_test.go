package templ

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestSprintt(t *testing.T) {
	c := qt.New(t)

	c.Assert(Sprintt("{{ . }}", "foo"), qt.Equals, "foo")
	c.Assert(Sprintt("{{ . | upper }}", "foo"), qt.Equals, "FOO")
	c.Assert(Sprintt("{{ . | lower }}", "FoO"), qt.Equals, "foo")
	c.Assert(Sprintt("{{ . | trimPrefix `v` }}", "v3.0.0"), qt.Equals, "3.0.0")
	c.Assert(Sprintt("{{ . | trimSuffix `-beta` }}", "v3.0.0-beta"), qt.Equals, "v3.0.0")
}
