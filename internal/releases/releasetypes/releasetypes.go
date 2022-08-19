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
