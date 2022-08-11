package external

import (
	"os"

	"github.com/bep/execrpc"
	"github.com/bep/execrpc/codecs"
	"github.com/bep/hugoreleaser/internal/archives"
	"github.com/bep/hugoreleaser/internal/archives/archiveformats"
	"github.com/bep/hugoreleaser/internal/config"
	"github.com/bep/hugoreleaser/pkg/plugins"
	"github.com/bep/hugoreleaser/pkg/plugins/archiveplugin"
	"github.com/bep/logg"
)

type GoRun struct {
}

func BuildArchive(infoLogger logg.LevelLogger, settings config.ArchiveSettings, req archiveplugin.Request) (err error) {
	if settings.Type.FormatParsed == archiveformats.External {
		return buildArchiveExternal(infoLogger, req)
	}

	outFile, err := os.Create(req.OutFilename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	archiver, err := archives.New(settings, outFile)
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

func buildArchiveExternal(infoLogger logg.LevelLogger, req archiveplugin.Request) error {
	infoLogger = infoLogger.WithField("plugin", "tar")

	client, err := execrpc.StartClient(
		execrpc.ClientOptions[archiveplugin.Request, archiveplugin.Response]{
			ClientRawOptions: execrpc.ClientRawOptions{
				Version: 1,
				Cmd:     "go",
				Args:    []string{"run", "."},
				Dir:     "./examples/plugins/archives/tar",

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

	defer client.Close() // TODO(bep)

	resp, err := client.Execute(req)
	if err != nil {
		return err
	}
	return resp.Err()
}
