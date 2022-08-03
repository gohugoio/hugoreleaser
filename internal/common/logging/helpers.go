package logging

import (
	"fmt"
	"os"
	"runtime"
	"time"

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

	fd := f.Fd()
	return os.Getenv("TERM") != "dumb" && (isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd))
}
