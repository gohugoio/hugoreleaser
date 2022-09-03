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

// Issue 30.
func TestGetVersionTagBefore(t *testing.T) {
	if isCI {
		// GitHub Actions clones shallowly, so we can't test this there.
		t.Skip("skip on CI")
	}
	c := qt.New(t)

	for _, test := range []struct {
		about  string
		ref    string
		expect string
	}{
		{
			about:  "patch tag",
			ref:    "v0.53.2",
			expect: "v0.53.1",
		},
		{
			about:  "patch commit",
			ref:    "8509f4591d37435df1bfb2bcb4dfb5fe474b0252",
			expect: "v0.53.2",
		},
		{
			about:  "minor",
			ref:    "v0.52.0",
			expect: "v0.51.0",
		},
	} {
		c.Run(test.about, func(c *qt.C) {
			tag, err := gitVersionTagBefore("", test.ref)
			c.Assert(err, qt.IsNil)
			c.Assert(tag, qt.Equals, test.expect)
		})
	}

}
