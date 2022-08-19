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

package model

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

type Initializer interface {
	// Init initializes a config struct, that could be parsing of strings into Go objects, compiling of Glob patterns etc.
	// It returns an error if the initialization failed.
	Init() error
}

type BuildContext struct {
	Project string `toml:"project"`
	Tag     string `toml:"ref"`
	Goos    string `toml:"goos"`
	Goarch  string `toml:"goarch"`
}

func NewBasicError(what string, err error) *BasicError {
	return &BasicError{Msg: fmt.Sprintf("%s: %v", what, err)}
}

// BasicError holds an error message.
type BasicError struct {
	Msg string `toml:"msg"`
}

func (r BasicError) Error() string {
	return r.Msg
}

// FromNMap converts m to T.
// See https://pkg.go.dev/github.com/mitchellh/mapstructure#section-readme
func FromMap[T any](m map[string]any) (T, error) {
	var t T
	err := mapstructure.WeakDecode(m, &t)
	return t, err
}
