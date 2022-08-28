package changelog

import (
	"os"
	"testing"

	qt "github.com/frankban/quicktest"
)

var isCI = os.Getenv("CI") != ""

func TestGetOps(t *testing.T) {
	if isCI {
		// GitHub Actions clones shallowly, so we can't test this there.
		t.Skip("skip on CI")
	}
	c := qt.New(t)

	tag, err := gitVersionTagBefore("", "v0.51.0")
	c.Assert(err, qt.IsNil)
	c.Assert(tag, qt.Equals, "v0.50.0")

	exists, err := gitTagExists("", "v0.50.0")
	c.Assert(err, qt.IsNil)
	c.Assert(exists, qt.Equals, true)

	exists, err = gitTagExists("", "v3.9.0")
	c.Assert(err, qt.IsNil)
	c.Assert(exists, qt.Equals, false)

	log, err := gitLog("", "v0.50.0", "v0.51.0", "main")
	c.Assert(err, qt.IsNil)
	c.Assert(log, qt.Contains, "Shuffle chunked builds")

	infos, err := gitLogToGitInfos(log)
	c.Assert(err, qt.IsNil)
	c.Assert(len(infos), qt.Equals, 3)

}
