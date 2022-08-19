package main

import (
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestReadMeTOC(t *testing.T) {
	c := qt.New(t)

	doc := `
	
## Heading 1

Some text.

### Headings 2

Some text.

## Headings 1-2

	
`

	toc, err := createToc(doc)
	c.Assert(err, qt.IsNil)

	c.Assert(strings.TrimSpace(toc), qt.Equals, "* [Heading 1](#heading-1)\n     * [Headings 2](#headings-2)\n * [Headings 1-2](#headings-1-2)")

}
