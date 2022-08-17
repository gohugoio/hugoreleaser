package plugins

import (
	"github.com/bep/execrpc"
	"github.com/bep/execrpc/codecs"
	"github.com/bep/hugoreleaser/internal/config"
	"github.com/bep/hugoreleaser/pkg/plugins"
	"github.com/bep/hugoreleaser/pkg/plugins/archiveplugin"
	"github.com/bep/logg"
)

// StartArchivePlugin starts an archive plugin.
func StartArchivePlugin(infoLogger logg.LevelLogger, options config.Plugin) (*execrpc.Client[archiveplugin.Request, archiveplugin.Response], error) {
	return execrpc.StartClient(
		execrpc.ClientOptions[archiveplugin.Request, archiveplugin.Response]{
			ClientRawOptions: execrpc.ClientRawOptions{
				Version: 1,
				Cmd:     "go",
				Args:    []string{"run", options.Command},
				Dir:     options.Dir,

				OnMessage: func(msg execrpc.Message) {
					statusCode := msg.Header.Status
					switch statusCode {
					case plugins.StatusInfoLog:
						infoLogger.Log(logg.String(string(msg.Body)))
					}
				},
			},
			Codec: codecs.TOMLCodec[archiveplugin.Request, archiveplugin.Response]{},
		},
	)

}
