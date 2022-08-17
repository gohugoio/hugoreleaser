package archives

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bep/hugoreleaser/cmd/corecmd"
	"github.com/bep/hugoreleaser/internal/archives/archiveformats"
	"github.com/bep/hugoreleaser/internal/config"
	"github.com/bep/hugoreleaser/pkg/plugins/archiveplugin"
	"github.com/bep/logg"
)

// Build builds an archive from the given settings and writes it to req.OutFilename
func Build(c *corecmd.Core, infoLogger logg.LevelLogger, settings config.ArchiveSettings, req archiveplugin.Request) (err error) {
	if settings.Type.FormatParsed == archiveformats.Plugin {
		// Delegate to external tool.
		return buildExternal(c, infoLogger, settings, req)
	}

	if c.Try {
		archive, err := New(settings, struct {
			io.Writer
			io.Closer
		}{
			io.Discard,
			io.NopCloser(nil),
		})
		if err != nil {
			return err
		}
		return archive.Finalize()
	}

	outFile, err := os.Create(req.OutFilename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	archiver, err := New(settings, outFile)
	if err != nil {
		return err
	}
	defer func() {
		err = archiver.Finalize()
	}()

	for _, file := range req.Files {
		f, err := os.Open(file.SourcePathAbs)
		if err != nil {
			return err
		}
		err = archiver.AddAndClose(file.TargetPath, f)
		if err != nil {
			return err
		}
	}

	return
}

func buildExternal(c *corecmd.Core, infoLogger logg.LevelLogger, settings config.ArchiveSettings, req archiveplugin.Request) error {
	infoLogger = infoLogger.WithField("plugin", settings.Plugin.Name)

	if c.Try {
		req.Heartbeat = fmt.Sprintf("heartbeat-%s", time.Now())
	}

	pluginSettings := settings.Plugin

	client, found := c.PluginsRegistryArchive[pluginSettings]
	if !found {
		return fmt.Errorf("archive plugin %q not found in registry", pluginSettings.Name)
	}

	resp, err := client.Execute(req)
	if err != nil {
		return err
	}

	if err := resp.Err(); err != nil {
		return err
	}

	if req.Heartbeat != resp.Heartbeat {
		// TODO(bep) make this into a typed error with a better error message further up.
		return fmt.Errorf("heartbeat mismatch: expected %q, got %q", req.Heartbeat, resp.Heartbeat)
	}

	return nil
}
