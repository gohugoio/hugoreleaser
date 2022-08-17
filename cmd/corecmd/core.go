package corecmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/bep/execrpc"
	"github.com/bep/hugoreleaser/internal/common/logging"
	"github.com/bep/hugoreleaser/internal/config"
	"github.com/bep/hugoreleaser/internal/plugins"
	"github.com/bep/hugoreleaser/pkg/plugins/archiveplugin"
	"github.com/bep/logg"
	"github.com/bep/workers"
	"github.com/pelletier/go-toml/v2"
	"github.com/peterbourgon/ff/v3/ffcli"
)

// CommandName is the main command's binary name.
const CommandName = "hugoreleaser"

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

	// The Git ref to build, e.g. a tag.
	Ref string

	// Abolute path to the project root.
	ProjectDir string

	// Absolute path to the dist directory.
	DistDir string

	// We store builds in ./dist/<project>/<ref>/<DistRootBuilds>/<os>/<arch>/<build
	DistRootBuilds string

	// We store archives in ./dist/<project>/<ref>/<DistRootArchives>/<os>/<arch>/<build
	DistRootArchives string

	// The config file to use.
	ConfigFile string

	// Number of parallel tasks.
	NumWorkers int

	// The global workforce.
	Workforce *workers.Workforce

	// Archive plugins started and ready to use.
	PluginsRegistryArchive map[config.Plugin]*execrpc.Client[archiveplugin.Request, archiveplugin.Response]

	// These will be set after Init() is sueccussfully executed.
	initDoneListeners []func()
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
	fs.StringVar(&c.Ref, "ref", "", "The Git ref to build.")
	fs.StringVar(&c.DistDir, "dist", "dist", "Directory to store the built artifacts in.")
	fs.StringVar(&c.ConfigFile, "config", "hugoreleaser.toml", "The config file to use.")
	fs.IntVar(&c.NumWorkers, "workers", 0, "Number of parallel builds.")
	fs.BoolVar(&c.Quiet, "quiet", false, "Don't output anything to stdout.")
	fs.BoolVar(&c.Try, "try", false, "Trial run, no builds, archives or releases.")
}

func (c *Core) AddInitDoneListener(cb func()) {
	c.initDoneListeners = append(c.initDoneListeners, cb)
}

func (c *Core) Init() error {
	var stdOut io.Writer
	if c.Quiet {
		stdOut = io.Discard
	} else {
		stdOut = os.Stdout
	}

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
		})

	// Configure logging.
	c.InfoLog = l.WithLevel(logg.LevelInfo).WithField("cmd", "core")
	c.WarnLog = l.WithLevel(logg.LevelWarn).WithField("cmd", "core")
	c.ErrorLog = l.WithLevel(logg.LevelError).WithField("cmd", "core")

	if c.Ref == "" {
		return fmt.Errorf("flag -ref is required")
	}

	// Set up the workers for parallel execution.
	if c.NumWorkers == 0 {
		c.NumWorkers = runtime.NumCPU()
	}

	c.Workforce = workers.New(c.NumWorkers)

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting working directory: %w", err)
	}

	c.ProjectDir = wd

	if !filepath.IsAbs(c.DistDir) {
		c.DistDir = filepath.Join(wd, c.DistDir)

		if err := os.MkdirAll(c.DistDir, 0o755); err != nil {
			return fmt.Errorf("error creating dist directory: %w", err)
		}
	}

	// These are not user-configurable.
	c.DistRootArchives = "archives"
	c.DistRootBuilds = "builds"

	if c.NumWorkers < 1 {
		c.NumWorkers = runtime.NumCPU()
	}

	if !filepath.IsAbs(c.ConfigFile) {
		c.ConfigFile = filepath.Join(wd, c.ConfigFile)
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

	// Start and register the archive plugins.
	c.PluginsRegistryArchive = make(map[config.Plugin]*execrpc.Client[archiveplugin.Request, archiveplugin.Response])

	startAndRegister := func(p config.Plugin) error {
		if p.IsZero() {
			return nil
		}
		if _, found := c.PluginsRegistryArchive[p]; found {
			// Already started.
			return nil
		}
		client, err := plugins.StartArchivePlugin(c.InfoLog, p)
		if err != nil {
			return fmt.Errorf("error starting archive plugin %q: %w", p.Name, err)
		}

		// Send a heartbeat to the plugin to make sure it's alive.
		heartbeat := fmt.Sprintf("heartbeat-%s", time.Now())
		resp, err := client.Execute(archiveplugin.Request{Heartbeat: heartbeat})
		if err != nil {
			return fmt.Errorf("error testing archive plugin %q: %w", p.Name, err)
		}
		if resp.Heartbeat != heartbeat {
			return fmt.Errorf("error testing archive plugin %q: unexpected heartbeat response", p.Name)
		}
		c.PluginsRegistryArchive[p] = client
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

	for _, l := range c.initDoneListeners {
		l()
	}

	return nil
}

func (c *Core) Close() error {
	for k, v := range c.PluginsRegistryArchive {
		if err := v.Close(); err != nil {
			c.WarnLog.Log(logg.String(fmt.Sprintf("error closing plugin %q: %s", k.Name, err)))
		}
	}
	return nil
}
