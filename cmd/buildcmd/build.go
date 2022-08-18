package buildcmd

import (
	"context"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/bep/helpers/envhelpers"

	"github.com/bep/hugoreleaser/cmd/corecmd"
	"github.com/bep/hugoreleaser/internal/builds"
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
	return &Builder{
		core: core,
	}
}

type Builder struct {
	core    *corecmd.Core
	infoLog logg.LevelLogger

	initOnce sync.Once
	initErr  error
}

func (b *Builder) Init() error {
	b.initOnce.Do(func() {
		b.infoLog = b.core.InfoLog.WithField("cmd", commandName)
	})
	return b.initErr
}

func (b *Builder) Exec(ctx context.Context, args []string) error {
	if err := b.Init(); err != nil {
		return err
	}

	buildsDistDir := filepath.Join(
		b.core.DistDir,
		b.core.Config.Project,
		b.core.Tag,
		b.core.DistRootBuilds,
	)

	// Remove and recreate the builds dist dir.
	_ = os.RemoveAll(buildsDistDir)
	if err := os.MkdirAll(buildsDistDir, 0o755); err != nil {
		return err
	}

	r, ctx := b.core.Workforce.Start(ctx)
	for _, build := range b.core.Config.Builds {
		for _, os := range build.Os {
			for _, arch := range os.Archs {
				// Capture this for the Go routine below.
				arch := arch
				r.Run(func() error {
					return b.buildArch(ctx, arch)
				})
			}
		}
	}
	return r.Wait()
}

func (b *Builder) buildArch(ctx context.Context, arch config.BuildArch) error {
	goexe := b.core.Config.BuildSettings.GoExe
	outFilename := filepath.Join(
		b.core.DistDir,
		b.core.Config.Project,
		b.core.Tag,
		b.core.DistRootBuilds,
		arch.Build.Path,
		arch.Os.Goos,
		arch.Goarch,
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
