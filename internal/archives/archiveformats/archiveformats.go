package archiveformats

import (
	"fmt"

	"github.com/bep/hugoreleaser/internal/common/mapsh"
)

// Goreleaser supports `tar.gz`, `tar.xz`, `tar`, `gz`, `zip` and `binary`.
// We currently limit ourselves to what Hugo uses: `tar.gz` and 'zip' (for Windows).
// TODO(bep) where does deb fit in?
const (
	InvalidFormat Format = iota
	Deb
	TarGz
	Zip
	Plugin // Plugin is a special format that is used to indicate that the archive operation is handled by an external tool.
)

var formatString = map[Format]string{
	// The string values is what users can specify in the config.
	Deb:    "deb",
	TarGz:  "tar.gz",
	Zip:    "zip",
	Plugin: "_plugin",
}

var stringFormat = map[string]Format{}

func init() {
	for k, v := range formatString {
		stringFormat[v] = k
	}
}

// ParseFormat parses a string into a Format.
func ParseFormat(s string) (Format, error) {
	f := stringFormat[s]
	if f == InvalidFormat {
		return f, fmt.Errorf("invalid archive format %q, must be one of %s", s, mapsh.KeysSorted(formatString))
	}
	return f, nil
}

// Format represents the type of archive.
type Format int

func (f Format) String() string {
	return formatString[f]
}
