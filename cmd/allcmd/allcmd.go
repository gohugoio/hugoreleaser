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

package allcmd

import (
	"context"
	"flag"

	"github.com/gohugoio/hugoreleaser/cmd/archivecmd"
	"github.com/gohugoio/hugoreleaser/cmd/buildcmd"
	"github.com/gohugoio/hugoreleaser/cmd/corecmd"
	"github.com/gohugoio/hugoreleaser/cmd/releasecmd"

	"github.com/bep/logg"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const commandName = "all"

// New returns a usable ffcli.Command for the archive subcommand.
func New(core *corecmd.Core) *ffcli.Command {
	fs := flag.NewFlagSet(corecmd.CommandName+" "+commandName, flag.ExitOnError)

	builder := buildcmd.NewBuilder(core, fs)
	a := &all{
		core:      core,
		builder:   builder,
		archivist: archivecmd.NewArchivist(core),
		releaser:  releasecmd.NewReleaser(core, fs),
	}

	core.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       commandName,
		ShortUsage: corecmd.CommandName + " " + commandName + " [flags] <action>",
		ShortHelp:  "Runs the commands build, archive and release in sequence.",
		FlagSet:    fs,
		Exec:       a.Exec,
	}
}

type all struct {
	infoLog logg.LevelLogger
	core    *corecmd.Core

	builder   *buildcmd.Builder
	archivist *archivecmd.Archivist
	releaser  *releasecmd.Releaser
}

func (a *all) Init() error {
	a.infoLog = a.core.InfoLog.WithField("all", commandName)
	return nil
}

func (a *all) Exec(ctx context.Context, args []string) error {
	if err := a.Init(); err != nil {
		return err
	}

	commandHandlers := []corecmd.CommandHandler{
		a.builder,
		a.archivist,
		a.releaser,
	}

	for _, commandHandler := range commandHandlers {
		if err := commandHandler.Init(); err != nil {
			return err
		}
	}

	for _, commandHandler := range commandHandlers {
		if err := commandHandler.Exec(ctx, args); err != nil {
			return err
		}
	}

	return nil
}
