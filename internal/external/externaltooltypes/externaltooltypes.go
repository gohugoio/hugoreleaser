package externaltooltypes

import (
	"fmt"

	"github.com/bep/hugoreleaser/internal/common/mapsh"
)

// Type is the type of external tool.
type Type int

func (t Type) String() string {
	return typeString[t]
}

const (
	// InvalidType is an invalid type.
	Invalid Type = iota

	// A external tool run via "go run ..."
	GoRun
)

// ParseType parses a string into a Type.
func ParseType(s string) (Type, error) {
	f := stringType[s]
	if f == Invalid {
		return f, fmt.Errorf("invalid tool type %q, must be one of %s", s, mapsh.KeysSorted(typeString))
	}
	return f, nil
}

var typeString = map[Type]string{
	// The string values is what users can specify in the config.
	GoRun: "gorun",
}

var stringType = map[string]Type{}

func init() {
	for k, v := range typeString {
		stringType[v] = k
	}
}
