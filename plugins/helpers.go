package plugins

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

// FromNMap converts m to T.
// See https://pkg.go.dev/github.com/mitchellh/mapstructure#section-readme
func FromMap[T any](m map[string]any) (T, error) {
	var t T
	err := mapstructure.WeakDecode(m, &t)
	return t, err
}

// Error is an error that can be returned from a plugin,
// it's main quality is that it can be marshalled to and from TOML/JSON etc.
func NewError(what string, err error) *Error {
	return &Error{Msg: fmt.Sprintf("%s: %v", what, err)}
}

// Error holds an error message.
type Error struct {
	Msg string `toml:"msg"`
}

func (r Error) Error() string {
	return r.Msg
}
