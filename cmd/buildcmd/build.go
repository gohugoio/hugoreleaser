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

package buildcmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bep/helpers/envhelpers"
	"github.com/gobwas/glob"

	"github.com/bep/hugoreleaser/cmd/corecmd"
	"github.com/bep/hugoreleaser/internal/builds"
	"github.com/bep/hugoreleaser/internal/common/ioh"
	"github.com/bep/hugoreleaser/internal/config"
	"github.com/bep/logg"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const commandName = "build"

// New returns a usable ffcli.Command for the build subcommand.
// TODO(bep): Add -paths (slice)
// TODO(bep): Add a coverage command (JSON?)
func New(core *corecmd.Core) *ffcli.Command {
	fs := flag.NewFlagSet(corecmd.CommandName+" "+commandName, flag.ExitOnError)
	core.RegisterFlags(fs)
	builder := NewBuilder(core, fs)

	return &ffcli.Command{
		Name:       "build",
		ShortUsage: corecmd.CommandName + " build [flags] <action>",
		ShortHelp:  "TODO(bep)",
		FlagSet:    fs,
		Exec:       builder.Exec,
	}
}

// NewBuilder returns a new Builder.
func NewBuilder(core *corecmd.Core, fs *flag.FlagSet) *Builder {
	b := &Builder{
		core:       core,
		BuildPaths: &BuildPaths{},
	}

	fs.StringVar(&b.BuildPaths.Paths, "build-paths", "/builds/**", "The builds to handle (defaults to all).")

	return b
}

type Builder struct {
	core    *corecmd.Core
	infoLog logg.LevelLogger

	BuildPaths *BuildPaths

	initOnce sync.Once
	initErr  error
}

type BuildPaths struct {
	Paths         string
	PathsCompiled glob.Glob

	initOnce sync.Once
}

func (b *BuildPaths) Init() error {
	var err error
	b.initOnce.Do(func() {
		const prefix = "/builds/"

		if !strings.HasPrefix(b.Paths, prefix) {
			err = fmt.Errorf("%s: flag -build-paths must start with %s", commandName, prefix)
			return
		}

		// Strip the /builds/ prefix. We currently don't use that,
		// it's just there to make the config easier to understand.
		paths := strings.TrimPrefix(b.Paths, prefix)

		b.PathsCompiled, err = glob.Compile(paths)

	})

	return err

}

func (b *Builder) Init() error {
	b.initOnce.Do(func() {
		b.infoLog = b.core.InfoLog.WithField("cmd", commandName)
		b.initErr = b.BuildPaths.Init()

	})
	return b.initErr
}

func (b *Builder) Exec(ctx context.Context, args []string) error {
	if err := b.Init(); err != nil {
		return err
	}

	r, ctx := b.core.Workforce.Start(ctx)

	for _, archPath := range b.core.Config.FindArchs(b.BuildPaths.PathsCompiled) {
		// Capture this for the Go routine below.
		archPath := archPath
		r.Run(func() error {
			return b.buildArch(ctx, archPath)
		})
	}

	return r.Wait()
}

func (b *Builder) buildArch(ctx context.Context, archPath config.BuildArchPath) error {
	goexe := b.core.Config.BuildSettings.GoExe
	arch := archPath.Arch
	outDir := filepath.Join(
		b.core.DistDir,
		b.core.Config.Project,
		b.core.Tag,
		b.core.DistRootBuilds,
		filepath.FromSlash(archPath.Path),
	)
	if err := ioh.RemoveAllMkdirAll(outDir); err != nil {
		return err
	}
	outFilename := filepath.Join(
		outDir,
		arch.BuildSettings.Binary,
	)

	b.infoLog.WithField("binary", outFilename).WithFields(b.core.Config.BuildSettings).Log(logg.String("Building"))

	if b.core.Try {
		return nil
	}

	buildSettings := arch.BuildSettings

	buildBinary := func(filename, goarch string) error {
		var keyVals []string
		args := []string{"build", "-o", filename}

		keyVals = append(
			keyVals,
			"GOOS", arch.Os.Goos,
			"GOARCH", goarch,
		)

		if arch.BuildSettings.Goarm != "" {
			keyVals = append(keyVals, "GOARM", arch.BuildSettings.Goarm)
		}

		if buildSettings.Env != nil {
			for _, env := range buildSettings.Env {
				key, val := envhelpers.SplitEnvVar(env)
				keyVals = append(keyVals, key, val)
			}
		}
		if buildSettings.GoProxy != "" {
			keyVals = append(keyVals, "GOPROXY", buildSettings.GoProxy)
		}

		environ := os.Environ()
		envhelpers.SetEnvVars(&environ, keyVals...)

		if buildSettings.Ldflags != "" {
			args = append(args, "-ldflags", buildSettings.Ldflags)
		}
		if buildSettings.Flags != nil {
			args = append(args, buildSettings.Flags...)
		}

		cmd := exec.CommandContext(ctx, goexe, args...)
		cmd.Env = environ
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		return cmd.Run()
	}

	if arch.Goarch == builds.UniversalGoarch {
		// Build for both arm64 and amd64 and then combine them into a universal binary.
		goarchs := []string{"arm64", "amd64"}
		var outFilenames []string
		for _, goarch := range goarchs {
			filename := outFilename + "_" + goarch
			outFilenames = append(outFilenames, filename)
			if err := buildBinary(filename, goarch); err != nil {
				return err
			}
		}
		if err := builds.CreateMacOSUniversalBinary(outFilename, outFilenames...); err != nil {
			return err
		}

		// Remove the individual binary files.
		for _, filename := range outFilenames {
			if err := os.Remove(filename); err != nil {
				return err
			}
		}

	} else {
		if err := buildBinary(outFilename, arch.Goarch); err != nil {
			return err
		}
	}

	return nil
}
