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

// Package logging contains some basic loggin setup.
package logging

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/bep/logg"

	"github.com/fatih/color"
)

// Default Handler implementation.
// Based on https://github.com/apex/log/blob/master/handlers/cli/cli.go
type DefaultHandler struct {
	mu        sync.Mutex
	outWriter io.Writer // Defaults to os.Stdout.
	errWriter io.Writer // Defaults to os.Stderr.

	Padding int
}

// NewDefaultHandler handler.
func NewDefaultHandler(outWriter, errWriter io.Writer) logg.Handler {
	return &DefaultHandler{
		outWriter: outWriter,
		errWriter: errWriter,
		Padding:   0,
	}
}

var bold = color.New(color.Bold)

// Colors mapping.
var Colors = [...]*color.Color{
	logg.LevelDebug: color.New(color.FgWhite),
	logg.LevelInfo:  color.New(color.FgBlue),
	logg.LevelWarn:  color.New(color.FgYellow),
	logg.LevelError: color.New(color.FgRed),
}

// Strings mapping.
var Strings = [...]string{
	logg.LevelDebug: "•",
	logg.LevelInfo:  "•",
	logg.LevelWarn:  "•",
	logg.LevelError: "⨯",
}

// HandleLog implements logg.Handler.
func (h *DefaultHandler) HandleLog(e *logg.Entry) error {
	color := Colors[e.Level]
	level := Strings[e.Level]

	h.mu.Lock()
	defer h.mu.Unlock()

	var w io.Writer
	if e.Level > logg.LevelInfo {
		w = h.errWriter
	} else {
		w = h.outWriter
	}

	const cmdName = "cmd"

	var prefix string
	for _, field := range e.Fields {
		if field.Name == cmdName {
			prefix = fmt.Sprint(field.Value)
			break
		}
	}

	if prefix != "" {
		prefix = strings.ToLower(prefix) + ": "
	}

	color.Fprintf(w, "%s %s%s", bold.Sprintf("%*s", h.Padding+1, level), color.Sprint(prefix), e.Message)

	for _, field := range e.Fields {
		if field.Name == cmdName {
			continue
		}
		fmt.Fprintf(w, " %s %v", color.Sprint(field.Name), field.Value)
	}

	fmt.Fprintln(w)

	return nil
}
