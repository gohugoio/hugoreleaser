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
	Ref     string `toml:"ref"`
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
