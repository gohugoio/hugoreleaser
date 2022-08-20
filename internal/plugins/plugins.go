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

package plugins

import (
	"strings"
	"time"

	"github.com/bep/execrpc"
	"github.com/bep/execrpc/codecs"
	"github.com/bep/hugoreleaser/internal/config"
	"github.com/bep/hugoreleaser/plugins"
	"github.com/bep/hugoreleaser/plugins/archiveplugin"
	"github.com/bep/logg"
)

// StartArchivePlugin starts a archive plugin.
func StartArchivePlugin(infoLogger logg.LevelLogger, goSettings config.GoSettings, options config.Plugin) (*execrpc.Client[archiveplugin.Request, archiveplugin.Response], error) {
	env := options.Env
	var hasGoProxy bool
	for _, e := range env {
		if strings.HasPrefix(e, "GOPROXY=") {
			hasGoProxy = true
			break
		}
	}
	if !hasGoProxy {
		env = append(env, "GOPROXY="+goSettings.GoProxy)
	}

	return execrpc.StartClient(
		execrpc.ClientOptions[archiveplugin.Request, archiveplugin.Response]{
			ClientRawOptions: execrpc.ClientRawOptions{
				Version: 1,
				Cmd:     goSettings.GoExe,
				Args:    []string{"run", options.Command},
				Dir:     options.Dir,
				Env:     env,
				Timeout: 220 * time.Second, // TODO(bep) make configurable + fix the GOMODCACHE on GitHub

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
