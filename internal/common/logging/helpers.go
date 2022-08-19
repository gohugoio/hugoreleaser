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
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/bep/logg"
	"github.com/mattn/go-isatty"
)

// FormatBuildDuration formats a duration to a string on the form expected in "Total in ..." etc.
func FormatBuildDuration(d time.Duration) string {
	if d.Milliseconds() < 2000 {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// IsTerminal return true if the file descriptor is terminal and the TERM
// environment variable isn't a dumb one.
func IsTerminal(f *os.File) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	if os.Getenv("CI") != "" {
		return true
	}

	fd := f.Fd()
	return os.Getenv("TERM") != "dumb" && (isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd))
}

// Replacer creates a new log handler that does string replacement in log messages.
func Replacer(repl *strings.Replacer) logg.Handler {
	return logg.HandlerFunc(func(e *logg.Entry) error {
		e.Message = repl.Replace(e.Message)
		for i, field := range e.Fields {
			if s, ok := field.Value.(string); ok {
				e.Fields[i].Value = repl.Replace(s)
			}
		}
		return nil
	})
}
