package matchers

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestGlob(t *testing.T) {
	c := qt.New(t)

	mustGlob := func(s ...string) Matcher {
		g, err := Glob(s...)
		c.Assert(err, qt.IsNil)
		c.Assert(g, qt.Not(qt.IsNil))
		return g
	}

	c.Assert(mustGlob("**").Match("a"), qt.Equals, true)
	c.Assert(mustGlob("foo").Match("foo"), qt.Equals, true)
	c.Assert(mustGlob("!foo").Match("foo"), qt.Equals, false)
	c.Assert(mustGlob("foo", "bar").Match("foo"), qt.Equals, false)

	c.Assert(mustGlob("builds/**").Match("builds/abc/def"), qt.Equals, true)

	_, err := Glob("")
	c.Assert(err, qt.ErrorMatches, "empty pattern")
}
