package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bep/helpers/envhelpers"
	"github.com/bep/helpers/filehelpers"
	"github.com/rogpeppe/go-internal/testscript"
)

func TestBasic(t *testing.T) {
	setup := testSetupFunc()
	testscript.Run(t, testscript.Params{
		Dir:      "testscripts/basic",
		TestWork: false,
		Setup: func(env *testscript.Env) error {
			return setup(env)
		},
	})
}

// Tests in development can be put in "testscripts/unfinished".
func TestUnfinished(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skip unfinished tests on CI")
	}

	setup := testSetupFunc()

	testscript.Run(t, testscript.Params{
		Dir:      "testscripts/unfinished",
		TestWork: false,
		//UpdateScripts: true,
		Setup: func(env *testscript.Env) error {
			return setup(env)
		},
	})
}

func testSetupFunc() func(env *testscript.Env) error {
	sourceDir, _ := os.Getwd()
	return func(env *testscript.Env) error {
		// SOURCE is where the hugoreleaser source code lives.
		// We do this so we can
		// 1. Copy the example/test plugins into the WORK dir where the test script is running.
		// 2. Append a replace directive to the plugins' go.mod to get the up-to-date version of the plugin API.
		//
		// This is a hybrid setup neeed to get a quick development cycle going.
		// In production, the plugin Go modules would be addressed on their full form, e.g. "github.com/bep/hugoreleaser/internal/plugins/archives/tar@v1.0.0".
		envhelpers.SetEnvVars(&env.Vars, "SOURCE", sourceDir)
		return nil
	}
}

func TestMain(m *testing.M) {

	os.Exit(
		testscript.RunMain(m, map[string]func() int{
			// The main program.
			"hugoreleaser": func() int {
				if err := parseAndRun(os.Args[1:]); err != nil {
					fmt.Fprintln(os.Stderr, err)
					return 1
				}
				return 0
			},

			// log prints to stderr.
			"log": func() int {
				log.Println(os.Args[1])
				return 0
			},
			"sleep": func() int {
				i, err := strconv.Atoi(os.Args[1])
				if err != nil {
					i = 1
				}
				time.Sleep(time.Duration(i) * time.Second)
				return 0
			},

			// cpdir copies a directory recursively.
			"cpdir": func() int {
				if len(os.Args) != 3 {
					fmt.Fprintln(os.Stderr, "usage: cpdir SRC DST")
					return 1
				}

				fromDir := os.Args[1]
				toDir := os.Args[2]

				if !filepath.IsAbs(fromDir) {
					fromDir = filepath.Join(os.Getenv("SOURCE"), fromDir)
				}

				err := filehelpers.CopyDir(fromDir, toDir, nil)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					return 1
				}
				return 0
			},

			// append appends to a file with a leaading newline.
			"append": func() int {
				if len(os.Args) < 3 {

					fmt.Fprintln(os.Stderr, "usage: append FILE TEXT")
					return 1
				}

				filename := os.Args[1]
				words := os.Args[2:]
				for i, word := range words {
					words[i] = strings.Trim(word, "\"")
				}
				text := strings.Join(words, " ")

				_, err := os.Stat(filename)
				if err != nil {
					if os.IsNotExist(err) {
						fmt.Fprintln(os.Stderr, "file does not exist:", filename)
						return 1
					}
					fmt.Fprintln(os.Stderr, err)
					return 1
				}

				f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					fmt.Fprintln(os.Stderr, "failed to open file:", filename)
					return 1
				}
				defer f.Close()

				_, err = f.WriteString("\n" + text)
				if err != nil {
					fmt.Fprintln(os.Stderr, "failed to write to file:", filename)
					return 1
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
