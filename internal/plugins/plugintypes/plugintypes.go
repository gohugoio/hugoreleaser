// Copyright 2022 The Hugoreleaser Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plugintypes

import (
	"fmt"
	"strings"

	"github.com/gohugoio/hugoreleaser/internal/common/mapsh"
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

// Parse parses a string into a Type.
func Parse(s string) (Type, error) {
	f := stringType[strings.ToLower(s)]
	if f == Invalid {
		return f, fmt.Errorf("invalid tool type %q, must be one of %s", s, mapsh.KeysSorted(typeString))
	}
	return f, nil
}

// MustParse is like Parse but panics if the string is not a valid type.
func MustParse(s string) Type {
	f, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return f
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
