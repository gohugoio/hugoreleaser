package plugintypes

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestType(t *testing.T) {
	c := qt.New(t)

	c.Assert(MustParse("gorun"), qt.Equals, GoRun)
	c.Assert(MustParse("GoRun").String(), qt.Equals, "gorun")

	_, err := Parse("invalid")
	c.Assert(err, qt.ErrorMatches, "invalid tool type \"invalid\", must be one of .*")
	c.Assert(func() { MustParse("invalid") }, qt.PanicMatches, `invalid.*`)

}
