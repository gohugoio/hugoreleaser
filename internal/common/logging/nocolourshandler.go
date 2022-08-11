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
		fmt.Fprintf(w, " %s %v", field.Name, field.Value)
	}
	fmt.Fprintln(w)

	return nil
}
