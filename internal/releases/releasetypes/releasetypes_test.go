package releasetypes

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestType(t *testing.T) {
	c := qt.New(t)

	c.Assert(MustParse("Github"), qt.Equals, GitHub)

	_, err := Parse("invalid")
	c.Assert(err, qt.ErrorMatches, "invalid release type \"invalid\", must be one of .*")
	c.Assert(func() { MustParse("invalid") }, qt.PanicMatches, `invalid.*`)

}
