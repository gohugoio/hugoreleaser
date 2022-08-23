package matchers

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

type stringMatcher string

func (m stringMatcher) Match(s string) bool {
	return s == string(m)
}

func TestOps(t *testing.T) {
	c := qt.New(t)

	c.Assert(And(stringMatcher("a"), stringMatcher("b")).Match("a"), qt.Equals, false)
	c.Assert(And(stringMatcher("a"), stringMatcher("a")).Match("a"), qt.Equals, true)
	c.Assert(And(stringMatcher("a"), stringMatcher("a"), stringMatcher("a")).Match("a"), qt.Equals, true)

	c.Assert(Or(stringMatcher("a"), stringMatcher("b")).Match("a"), qt.Equals, true)
	c.Assert(Or(stringMatcher("a"), stringMatcher("b")).Match("c"), qt.Equals, false)

	c.Assert(Not(stringMatcher("a")).Match("a"), qt.Equals, false)
	c.Assert(Not(stringMatcher("a")).Match("b"), qt.Equals, true)

	c.Assert(MatchEverything.Match("a"), qt.Equals, true)

}
