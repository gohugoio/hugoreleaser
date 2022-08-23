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

package corecmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/bep/execrpc"
	"github.com/bep/helpers/envhelpers"
	"github.com/bep/logg"
	"github.com/bep/logg/handlers/multi"
	"github.com/bep/workers"
	"github.com/gohugoio/hugoreleaser/internal/common/logging"
	"github.com/gohugoio/hugoreleaser/internal/common/matchers"
	"github.com/gohugoio/hugoreleaser/internal/common/templ"
	"github.com/gohugoio/hugoreleaser/internal/config"
	"github.com/gohugoio/hugoreleaser/internal/plugins"
	"github.com/gohugoio/hugoreleaser/plugins/archiveplugin"
	"github.com/gohugoio/hugoreleaser/plugins/model"
	"github.com/pelletier/go-toml/v2"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type CommandHandler interface {
	Exec(ctx context.Context, args []string) error
	Init() error
}

const (

	// CommandName is the main command's binary name.
	CommandName = "hugoreleaser"

	// The prefix used for any flag overrides.
	EnvPrefix = "HUGORELEASER"

	// The env file to look for in the current directory.
	EnvFile = "hugoreleaser.env"
)

// New constructs a usable ffcli.Command and an empty Config. The config
// will be set after a successful parse. The caller must
func New() (*ffcli.Command, *Core) {
	var cfg Core

	fs := flag.NewFlagSet(CommandName, flag.ExitOnError)

	cfg.RegisterFlags(fs)

	return &ffcli.Command{
		Name:       CommandName,
		ShortUsage: CommandName + " [flags] <subcommand> [flags] [<arg>...]",
		FlagSet:    fs,
		Exec:       cfg.Exec,
	}, &cfg
}

// Core holds common config settings and objects.
type Core struct {
	// The parsed config.
	Config config.Config

	// The common Info logger.
	InfoLog logg.LevelLogger

	// The common Warn logger.
	WarnLog logg.LevelLogger

	// The common Error logger.
	ErrorLog logg.LevelLogger

	// No output to stdout.
	Quiet bool

	// Trial run, no builds or releases.
	Try bool

	// The Git tag to use for the release.
	// This tag will eventually be created at release time if it does not exist.
	Tag string

	// Paths to build/release.
	Paths                 stringFlags
	PathsBuildsCompiled   matchers.Matcher
	PathsArchivesCompiled matchers.Matcher
	PathsReleasesCompiled matchers.Matcher

	// Abolute path to the project root.
	ProjectDir string

	// Absolute path to the dist directory.
	DistDir string

	// We store builds in ./dist/<project>/<ref>/<DistRootBuilds>/<os>/<arch>/<build
	DistRootBuilds string

	// We store archives in ./dist/<project>/<ref>/<DistRootArchives>/<os>/<arch>/<build
	DistRootArchives string

	// We store release artifacts in ./dist/<project>/<ref>/<DistRootReleases>/<release.dir>
	DistRootReleases string

	// The config file to use.
	ConfigFile string

	// Number of parallel tasks.
	NumWorkers int

	// Global timeout for all commands.
	Timeout time.Duration

	// The global workforce.
	Workforce *workers.Workforce

	// Archive plugins started and ready to use.
	PluginsRegistryArchive map[string]*execrpc.Client[archiveplugin.Request, archiveplugin.Response]
}

// Exec function for this command.
func (c *Core) Exec(context.Context, []string) error {
	// The root command has no meaning, so if it gets executed,
	// display the usage text to the user instead.
	return flag.ErrHelp
}

// RegisterFlags registers the flag fields into the provided flag.FlagSet. This
// helper function allows subcommands to register the root flags into their
// flagsets, creating "global" flags that can be passed after any subcommand at
// the commandline.
func (c *Core) RegisterFlags(fs *flag.FlagSet) {
	numWorkers := runtime.NumCPU()
	if numWorkers > 6 {
		numWorkers = 6
	}
	fs.StringVar(&c.Tag, "tag", "", "The name of the release tag (e.g. v1.2.0). Does not need to exist.")
	fs.Var(&c.Paths, "paths", "Paths to include in the command.")
	fs.StringVar(&c.DistDir, "dist", "dist", "Directory to store the built artifacts in.")
	fs.StringVar(&c.ConfigFile, "config", "hugoreleaser.toml", "The config file to use.")
	fs.IntVar(&c.NumWorkers, "workers", numWorkers, "Number of parallel builds.")
	fs.DurationVar(&c.Timeout, "timeout", 55*time.Minute, "Global timeout.")
	fs.BoolVar(&c.Quiet, "quiet", false, "Don't output anything to stdout.")
	fs.BoolVar(&c.Try, "try", false, "Trial run, no builds, archives or releases.")

}

// PreInit is called before the flags are parsed.
func (c *Core) PreInit() error {
	// We need to do this as early as possible (before the flags and config is parsed).
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting working directory: %w", err)
	}

	c.ProjectDir = wd

	// Note that OS env will override config env.
	if env, err := config.LoadEnvFile(filepath.Join(c.ProjectDir, EnvFile)); err == nil {
		for k, v := range env {
			if os.Getenv(k) == "" {
				os.Setenv(k, v)
			}
		}
	}

	return nil
}

func (c *Core) compilePaths() error {
	// First check if it should match everything (default).
	shouldMatchEverything := true
	for _, p := range c.Paths {
		if strings.HasPrefix(p, "!") {
			shouldMatchEverything = false
			break
		}
		if strings.HasPrefix(p, "/") {
			return fmt.Errorf("paths must not start with /: %s", p)
		}
		if p != "**" {
			shouldMatchEverything = false
			break
		}
	}
	if shouldMatchEverything {
		c.PathsBuildsCompiled = matchers.MatchEverything
		c.PathsArchivesCompiled = matchers.MatchEverything
		c.PathsReleasesCompiled = matchers.MatchEverything
		return nil
	}

	const (
		buildsPrefix   = "builds/"
		archivesPrefix = "archives/"
		releasesPrefix = "releases/"
	)

	compilePrefix := func(target *matchers.Matcher, p, prefix string) error {
		if !strings.HasPrefix(p, prefix) {
			return nil
		}
		// We remove the prefix from the pattern, which
		// makes later matching easier.
		// We may need the distinction at some point, but not now
		p = p[len(prefix):]
		pc, err := matchers.Glob(p)
		if err != nil {
			return fmt.Errorf("error compiling path %q: %w", p, err)
		}
		if *target == nil {
			*target = pc
		} else {
			*target = matchers.And(*target, pc)
		}
		return nil
	}

	// Specific paths needs to start with either builds/, archives/ or releases/.
	for _, p := range c.Paths {
		if !strings.HasPrefix(p, buildsPrefix) && !strings.HasPrefix(p, archivesPrefix) && !strings.HasPrefix(p, releasesPrefix) {
			return fmt.Errorf("path %q must start with builds/ or archives/ or releases/", p)
		}

		err := compilePrefix(&c.PathsBuildsCompiled, p, buildsPrefix)
		if err != nil {
			return err
		}
		err = compilePrefix(&c.PathsArchivesCompiled, p, archivesPrefix)
		if err != nil {
			return err
		}
		err = compilePrefix(&c.PathsReleasesCompiled, p, releasesPrefix)
		if err != nil {
			return err
		}
	}

	if c.PathsBuildsCompiled == nil {
		c.PathsBuildsCompiled = matchers.MatchEverything
	}
	if c.PathsArchivesCompiled == nil {
		c.PathsArchivesCompiled = matchers.MatchEverything
	}
	if c.PathsReleasesCompiled == nil {
		c.PathsReleasesCompiled = matchers.MatchEverything
	}

	return nil
}

func (c *Core) Init() error {
	var stdOut io.Writer
	if c.Quiet {
		stdOut = io.Discard
	} else {
		stdOut = os.Stdout
	}

	// Configure logging.
	var logHandler logg.Handler
	if logging.IsTerminal(os.Stdout) {
		logHandler = logging.NewDefaultHandler(stdOut, os.Stderr)
	} else {
		logHandler = logging.NewNoColoursHandler(stdOut, os.Stderr)
	}

	l := logg.New(
		logg.Options{
			Level:   logg.LevelInfo,
			Handler: logHandler,
		},
	)

	c.InfoLog = l.WithLevel(logg.LevelInfo).WithField("cmd", "init")

	if !filepath.IsAbs(c.DistDir) {
		c.DistDir = filepath.Join(c.ProjectDir, c.DistDir)

		if err := os.MkdirAll(c.DistDir, 0o755); err != nil {
			return fmt.Errorf("error creating dist directory: %w", err)
		}
	}

	c.InfoLog.WithField("directory", c.DistDir).Log(logg.String("Writing files to"))

	logHandler = multi.New(
		// Replace the Dist dir (usually long path) in the log messages with a shorter version.
		logging.Replacer(strings.NewReplacer(c.DistDir, "$DIST")), logHandler,
	)

	l = logg.New(
		logg.Options{
			Level:   logg.LevelInfo,
			Handler: logHandler,
		},
	)

	c.InfoLog = l.WithLevel(logg.LevelInfo).WithField("cmd", "core")
	c.WarnLog = l.WithLevel(logg.LevelWarn).WithField("cmd", "core")
	c.ErrorLog = l.WithLevel(logg.LevelError).WithField("cmd", "core")

	if c.Tag == "" {
		return fmt.Errorf("flag -tag is required")
	}

	if err := c.compilePaths(); err != nil {
		return fmt.Errorf("error compiling -paths: %w", err)
	}

	c.Workforce = workers.New(c.NumWorkers)

	// These are not user-configurable.
	c.DistRootArchives = "archives"
	c.DistRootBuilds = "builds"
	c.DistRootReleases = "releases"

	if c.NumWorkers < 1 {
		c.NumWorkers = runtime.NumCPU()
	}

	if !filepath.IsAbs(c.ConfigFile) {
		c.ConfigFile = filepath.Join(c.ProjectDir, c.ConfigFile)
	}

	f, err := os.Open(c.ConfigFile)
	if err != nil {
		return fmt.Errorf("error opening config file %q: %w", c.ConfigFile, err)
	}
	defer f.Close()

	c.Config, err = config.DecodeAndApplyDefaults(f)

	if err != nil {
		msg := "error decoding config file"
		switch v := err.(type) {
		case *toml.DecodeError:
			line, col := v.Position()
			return fmt.Errorf("%s %q:%d:%d %w:\n%s", msg, c.ConfigFile, line, col, err, v.String())
		case *toml.StrictMissingError:
			return fmt.Errorf("%s %q: %w:\n%s", msg, c.ConfigFile, err, v.String())
		}
		return fmt.Errorf("%s %q: %w", msg, c.ConfigFile, err)
	}

	// Precompile the common navigation for all archives.
	for i, archive := range c.Config.Archives {
		archiveSettings := archive.ArchiveSettings
		archs := c.Config.FindArchs(archive.PathsCompiled)
		for _, archPath := range archs {
			arch := archPath.Arch
			buildInfo := model.BuildInfo{
				Project: c.Config.Project,
				Tag:     c.Tag,
				Goos:    arch.Os.Goos,
				Goarch:  arch.Goarch,
			}
			name, err := templ.Sprintt(archive.ArchiveSettings.NameTemplate, buildInfo)
			if err != nil {
				return fmt.Errorf("error compiling archive name template: %w", err)
			}
			name = archiveSettings.ReplacementsCompiled.Replace(name) + archiveSettings.Type.Extension
			archPath.Name = name
			c.Config.Archives[i].ArchsCompiled = append(c.Config.Archives[i].ArchsCompiled, archPath)
		}
	}

	// Start and register the archive plugins.
	c.PluginsRegistryArchive = make(map[string]*execrpc.Client[archiveplugin.Request, archiveplugin.Response])

	startAndRegister := func(p config.Plugin) error {
		if p.IsZero() {
			return nil
		}
		if _, found := c.PluginsRegistryArchive[p.ID]; found {
			// Already started.
			return nil
		}
		infoCtx := c.InfoLog.WithField("plugin", p.ID)
		client, err := plugins.StartArchivePlugin(c.InfoLog, c.Config.GoSettings, p)
		if err != nil {
			return fmt.Errorf("error starting archive plugin %q: %w", p.ID, err)
		}

		// Send a heartbeat to the plugin to make sure it's alive.
		heartbeat := fmt.Sprintf("heartbeat-%s", time.Now())
		resp, err := client.Execute(archiveplugin.Request{Heartbeat: heartbeat})
		if err != nil {
			return fmt.Errorf("error testing archive plugin %q: %w", p.ID, err)
		}
		if resp.Heartbeat != heartbeat {
			return fmt.Errorf("error testing archive plugin %q: unexpected heartbeat response", p.ID)
		}
		infoCtx.Log(logg.String("Archive plugin started and ready for use"))
		c.PluginsRegistryArchive[p.ID] = client
		return nil
	}

	if err := startAndRegister(c.Config.ArchiveSettings.Plugin); err != nil {
		return err
	}
	for _, archive := range c.Config.Archives {
		if err := startAndRegister(archive.ArchiveSettings.Plugin); err != nil {
			return err
		}
	}

	return nil
}

func (c *Core) Close() error {
	for k, v := range c.PluginsRegistryArchive {
		if err := v.Close(); err != nil {
			if !errors.Is(err, execrpc.ErrShutdown) {
				c.WarnLog.Log(logg.String(fmt.Sprintf("error closing plugin %q: %s", k, err)))
			}
		}
	}
	return nil
}

func (c *Core) RunGo(ctx context.Context, envKeyVals, args []string, stderr io.Writer) error {
	buildSettings := c.Config.BuildSettings
	goexe := buildSettings.GoSettings.GoExe
	envKeyVals = append(envKeyVals, "GOPROXY", buildSettings.GoSettings.GoProxy)
	environ := os.Environ()
	envhelpers.SetEnvVars(&environ, envKeyVals...)
	cmd := exec.CommandContext(ctx, goexe, args...)
	cmd.Env = environ
	cmd.Stderr = stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

type stringFlags []string

func (s *stringFlags) String() string {
	return strings.Join(*s, "  ")
}

func (s *stringFlags) Set(value string) error {
	*s = append(*s, value)
	return nil
}
