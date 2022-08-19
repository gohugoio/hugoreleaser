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

package logging

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/bep/logg"
)

type NoColoursHandler struct {
	mu        sync.Mutex
	outWriter io.Writer // Defaults to os.Stdout.
	errWriter io.Writer // Defaults to os.Stderr.
}

// NewNoColoursHandler creates a new NoColoursHandler
func NewNoColoursHandler(outWriter, errWriter io.Writer) *NoColoursHandler {
	return &NoColoursHandler{
		outWriter: outWriter,
		errWriter: errWriter,
	}
}

func (h *NoColoursHandler) HandleLog(e *logg.Entry) error {
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
		prefix = strings.ToUpper(prefix) + ":\t"
	}

	fmt.Fprintf(w, "%s%s", prefix, e.Message)
	for _, field := range e.Fields {
		if field.Name == cmdName {
			continue
		}
		fmt.Fprintf(w, " %s %q", field.Name, field.Value)
	}
	fmt.Fprintln(w)

	return nil
}
