package logging

import (
	"fmt"
	"io"
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

	fmt.Fprintf(w, "%s", e.Message)
	for _, field := range e.Fields {
		if field.Name == "cmd" {
			continue
		}
		fmt.Fprintf(w, " %s %v", field.Name, field.Value)
	}
	fmt.Fprintln(w)

	return nil
}
