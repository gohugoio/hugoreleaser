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
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bep/helpers/envhelpers"
	"github.com/bep/helpers/slicehelpers"

	"github.com/bep/logg"
	"github.com/gohugoio/hugoreleaser/cmd/corecmd"
	"github.com/gohugoio/hugoreleaser/internal/builds"
	"github.com/gohugoio/hugoreleaser/internal/config"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const commandName = "build"

// New returns a usable ffcli.Command for the build subcommand.
func New(core *corecmd.Core) *ffcli.Command {
	fs := flag.NewFlagSet(corecmd.CommandName+" "+commandName, flag.ExitOnError)
	core.RegisterFlags(fs)
	builder := NewBuilder(core, fs)

	return &ffcli.Command{
		Name:       "build",
		ShortUsage: corecmd.CommandName + " build [flags] <action>",
		ShortHelp:  "Build Go binaries.",
		FlagSet:    fs,
		Exec:       builder.Exec,
	}
}

// NewBuilder returns a new Builder.
func NewBuilder(core *corecmd.Core, fs *flag.FlagSet) *Builder {
	b := &Builder{
		core: core,
	}

	fs.IntVar(&b.chunks, "chunks", -1, "Number of chunks to split the build into (optional).")
	fs.IntVar(&b.chunkIndex, "chunk-index", -1, "Index of the chunk to build (optional).")

	return b
}

type Builder struct {
	core    *corecmd.Core
	infoLog logg.LevelLogger

	chunks     int
	chunkIndex int
}

func (b *Builder) Init() error {
	b.infoLog = b.core.InfoLog.WithField("cmd", commandName)

	if b.chunks > 0 && b.chunkIndex >= b.chunks {
		return fmt.Errorf("chunk-index (%d) must be less than chunks (%d)", b.chunkIndex, b.chunks)
	}
	if b.chunks > 0 && b.chunkIndex < 0 {
		return fmt.Errorf("chunks (%d) requires chunk-index to be set", b.chunks)
	}

	return nil
}

func (b *Builder) Exec(ctx context.Context, args []string) error {
	if err := b.Init(); err != nil {
		return err
	}

	if !b.core.Try {
		// Prevarm the GOMODCACHE if this is a Go module project.
		if _, err := os.Stat(filepath.Join(b.core.ProjectDir, "go.mod")); err == nil {
			b.infoLog.Log(logg.String("Running 'go mod download'."))
			var buff bytes.Buffer
			if err := b.core.RunGo(ctx, nil, []string{"mod", "download"}, &buff); err != nil {
				b.core.ErrorLog.Log(logg.String(buff.String()))
				return err
			}
		}
	}

	archs := b.core.Config.FindArchs(b.core.PathsBuildsCompiled)

	if b.chunks > 0 {
		partitions := slicehelpers.Chunk(archs, b.chunks)
		if len(partitions) <= b.chunkIndex {
			archs = nil
			b.infoLog.Logf("No GOOS/GOARCHs available for chunk %d of %d.", b.chunkIndex+1, b.chunks)
		} else {
			archs = partitions[b.chunkIndex]
			b.infoLog.Logf("Building %d GOOS/GOARCHs in chunk %d of %d.", len(archs), b.chunkIndex+1, b.chunks)
		}
	} else {
		b.infoLog.Logf("Building %d GOOS/GOARCHs.", len(archs))
	}

	if len(archs) == 0 {
		return nil
	}

	r, ctx := b.core.Workforce.Start(ctx)

	for _, archPath := range archs {
		// Capture this for the Go routine below.
		archPath := archPath
		r.Run(func() error {
			return b.buildArch(ctx, archPath)
		})
	}

	return r.Wait()
}

func (b *Builder) buildArch(ctx context.Context, archPath config.BuildArchPath) error {
	arch := archPath.Arch
	outDir := filepath.Join(
		b.core.DistDir,
		b.core.Config.Project,
		b.core.Tag,
		b.core.DistRootBuilds,
		filepath.FromSlash(archPath.Path),
	)

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

		if buildSettings.Env != nil {
			for _, env := range buildSettings.Env {
				key, val := envhelpers.SplitEnvVar(env)
				keyVals = append(keyVals, key, val)
			}
		}

		if buildSettings.Ldflags != "" {
			args = append(args, "-ldflags", buildSettings.Ldflags)
		}
		if buildSettings.Flags != nil {
			args = append(args, buildSettings.Flags...)
		}

		return b.core.RunGo(ctx, keyVals, args, os.Stderr)
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
		b.infoLog.Logf("Combining %v into a universal binary.", goarchs)
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
