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

type Initializer interface {
	// Init initializes a config struct, that could be parsing of strings into Go objects, compiling of Glob patterns etc.
	// It returns an error if the initialization failed.
	Init() error
}

// BuildContext hold the core information about a build.
// This is used both by the plugins and the built-in archive implementation
// as the template context for the name template.
// It's kept here in this package because it's convenient.
type BuildContext struct {
	Project string `toml:"project"`
	Tag     string `toml:"tag"`
	Goos    string `toml:"goos"`
	Goarch  string `toml:"goarch"`
}
