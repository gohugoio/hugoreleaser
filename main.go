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
	"github.com/bep/hugoreleaser/cmd/releasecmd"
	"github.com/bep/hugoreleaser/internal/common/logging"
	"github.com/bep/logg"
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
		releasecmd        = releasecmd.New(core)
	)

	coreCommand.Subcommands = []*ffcli.Command{
		buildCommand,
		archiveCommand,
		releasecmd,
	}

	defer func() {
		if closeErr := core.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("error closing app: %w", err)
		}
		elapsed := time.Since(start)
		s := logg.String(fmt.Sprintf("Total in %s …", logging.FormatBuildDuration(elapsed)))
		if core.InfoLog != nil {
			core.InfoLog.Log(s)
		} else {
			log.Print(s)
		}
	}()

	if err := coreCommand.Parse(args); err != nil {
		return fmt.Errorf("error parsing command line: %w", err)
	}

	if err := core.Init(); err != nil {
		return fmt.Errorf("error initializing config: %w", err)
	}

	// TODO(bep) add a global timeout.
	if err := coreCommand.Run(context.Background()); err != nil {
		return fmt.Errorf("error running command: %w", err)
	}

	return
}
