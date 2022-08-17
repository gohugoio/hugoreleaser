package archives

import (
	"os"

	"github.com/bep/execrpc"
	"github.com/bep/execrpc/codecs"
	"github.com/bep/hugoreleaser/internal/archives/archiveformats"
	"github.com/bep/hugoreleaser/internal/config"
	"github.com/bep/hugoreleaser/pkg/plugins"
	"github.com/bep/hugoreleaser/pkg/plugins/archiveplugin"
	"github.com/bep/logg"
)

// Build builds an archive from the given settings and writes it to req.OutFilename
func Build(infoLogger logg.LevelLogger, settings config.ArchiveSettings, req archiveplugin.Request) (err error) {
	if settings.Type.FormatParsed == archiveformats.External {
		// Delegate to external tool.
		return buildExternal(infoLogger, settings, req)
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

func buildExternal(infoLogger logg.LevelLogger, settings config.ArchiveSettings, req archiveplugin.Request) error {
	infoLogger = infoLogger.WithField("plugin", "tar")

	pluginSettings := settings.Plugin

	client, err := execrpc.StartClient(
		execrpc.ClientOptions[archiveplugin.Request, archiveplugin.Response]{
			ClientRawOptions: execrpc.ClientRawOptions{
				Version: 1,
				Cmd:     "go",
				Args:    []string{"run", pluginSettings.Command},
				Dir:     pluginSettings.Dir,

				OnMessage: func(msg execrpc.Message) {
					statusCode := msg.Header.Status
					switch statusCode {
					case plugins.StatusInfoLog:
						infoLogger.Log(logg.String(string(msg.Body)))
					}
				},
			},
			Codec: codecs.JSONCodec[archiveplugin.Request, archiveplugin.Response]{},
		},
	)

	if err != nil {
		return err
	}

	defer client.Close() // TODO(bep) consider making these live for the whole build process.

	resp, err := client.Execute(req)
	if err != nil {
		return err
	}
	return resp.Err()
}
