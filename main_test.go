package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestBasic(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testscripts/basic",
	})
}

// Tests in development can be put in "testscripts/unfinished".
func TestUnfinished(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skip unfinished tests on CI")
	}
	testscript.Run(t, testscript.Params{
		Dir:      "testscripts/unfinished",
		TestWork: false,
	})
}

func TestMain(m *testing.M) {
	os.Exit(
		testscript.RunMain(m, map[string]func() int{
			// The main program.
			"hugoreleaser": func() int {
				if err := parseAndRun(os.Args[1:]); err != nil {
					fmt.Fprintln(os.Stderr, err)
					return -1
				}
				return 0
			},

			// Helpers.
			"checkfile": func() int {
				// The built-in exists does not check for zero size files.
				args := os.Args[1:]
				var readonly, exec bool
			loop:
				for len(args) > 0 {
					switch args[0] {
					case "-readonly":
						readonly = true
						args = args[1:]
					case "-exec":
						exec = true
						args = args[1:]
					default:
						break loop
					}
				}
				if len(args) == 0 {
					fatalf("usage: checkfile [-readonly] [-exec] file...")
				}

				for _, filename := range args {

					fi, err := os.Stat(filename)
					if err != nil {
						fmt.Fprintf(os.Stderr, "stat %s: %v\n", filename, err)
						return -1
					}
					if fi.Size() == 0 {
						fmt.Fprintf(os.Stderr, "%s is empty\n", filename)
						return -1
					}
					if readonly && fi.Mode()&0o222 != 0 {
						fmt.Fprintf(os.Stderr, "%s is writable\n", filename)
						return -1
					}
					if exec && runtime.GOOS != "windows" && fi.Mode()&0o111 == 0 {
						fmt.Fprintf(os.Stderr, "%s is not executable\n", filename)
						return -1
					}
				}

				return 0
			},
			"gobinary": func() int {
				if runtime.GOOS == "windows" {
					// TODO(bep) I assume this just doesn't work on Windows.
					return 0
				}
				if len(os.Args) < 3 {
					fatalf("usage: gobinary binary args...")
				}

				filename := os.Args[1]
				pattern := os.Args[2]
				if !strings.HasPrefix(pattern, "(") {
					// Multiline matching.
					pattern = "(?s)" + pattern
				}
				re := regexp.MustCompile(pattern)

				cmd := exec.Command("go", "version", "-m", filename)
				cmd.Stderr = os.Stderr

				b, err := cmd.Output()
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					return -1
				}

				output := string(b)

				if !re.MatchString(output) {
					fmt.Fprintf(os.Stderr, "expected %q to match %q\n", output, re)
					return -1
				}

				return 0
			},
		}),
	)
}

func fatalf(format string, a ...any) {
	panic(fmt.Sprintf(format, a...))
}
