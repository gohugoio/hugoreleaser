package releasetypes

import (
	"fmt"

	"github.com/bep/hugoreleaser/internal/common/mapsh"
)

type Type int

const (
	InvalidType Type = iota
	GitHub
)

var releaseTypeString = map[Type]string{
	GitHub: "github",
}

var stringReleaseType = map[string]Type{}

func init() {
	for k, v := range releaseTypeString {
		stringReleaseType[v] = k
	}
}

func (t Type) String() string {
	return releaseTypeString[t]
}

// Parse parses a string into a ReleaseType.
func Parse(s string) (Type, error) {
	t := stringReleaseType[s]
	if t == InvalidType {
		return t, fmt.Errorf("invalid release type %q, must be one of %s", s, mapsh.KeysSorted(releaseTypeString))
	}
	return t, nil
}
