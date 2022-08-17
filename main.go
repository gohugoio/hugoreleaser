package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"time"

	"github.com/bep/hugoreleaser/cmd/archivecmd"
	"github.com/bep/hugoreleaser/cmd/buildcmd"
	"github.com/bep/hugoreleaser/cmd/corecmd"
	"github.com/bep/hugoreleaser/internal/common/logging"
	"github.com/bep/logg"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func main() {
	if err := parseAndRun(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func parseAndRun(args []string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
			err = fmt.Errorf("%v", r)
		}
	}()

	start := time.Now()

	var (
		coreCommand, core = corecmd.New()
		buildCommand      = buildcmd.New(core)
		archiveCommand    = archivecmd.New(core)
	)

	coreCommand.Subcommands = []*ffcli.Command{
		buildCommand,
		archiveCommand,
	}

	coreCommand.Options = []ff.Option{
		ff.WithEnvVarPrefix("HUGORELEASER"),
	}

	defer func() {
		if err = core.Close(); err != nil {
			err = fmt.Errorf("error closing app: %w", err)
		}

		elapsed := time.Since(start)
		core.InfoLog.Log(logg.String(fmt.Sprintf("Total in %s …", logging.FormatBuildDuration(elapsed))))
	}()

	if err := coreCommand.Parse(args); err != nil {
		return fmt.Errorf("error parsing command line: %w", err)
	}

	if err := core.Init(); err != nil {
		return fmt.Errorf("error initializing config: %w", err)
	}

	if err := coreCommand.Run(context.Background()); err != nil {
		return fmt.Errorf("error running command: %w", err)
	}

	return
}
