package archiveformats

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestFormat(t *testing.T) {
	c := qt.New(t)

	c.Assert(MustParse("tar.gz"), qt.Equals, TarGz)
	c.Assert(MustParse("tar.gz").String(), qt.Equals, "tar.gz")
	c.Assert(MustParse("ZiP").String(), qt.Equals, "zip")

	_, err := Parse("invalid")
	c.Assert(err, qt.ErrorMatches, "invalid archive format \"invalid\", must be one of .*")
	c.Assert(func() { MustParse("invalid") }, qt.PanicMatches, `invalid.*`)

}
