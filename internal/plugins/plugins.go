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
	"github.com/bep/logg"
	"github.com/gohugoio/hugoreleaser-plugins-api/archiveplugin"
	"github.com/gohugoio/hugoreleaser-plugins-api/model"
	"github.com/gohugoio/hugoreleaser/internal/config"
)

// This represents a major version.
// Increment this only when there are breaking changes to the plugin protocol.
const pluginProtocolVersion = 2

type ArchivePluginConfig struct {
	Infol      logg.LevelLogger
	Try        bool
	GoSettings config.GoSettings
	Options    config.Plugin
	Project    string
	Tag        string
}

// StartArchivePlugin starts a archive plugin.
func StartArchivePlugin(cfg ArchivePluginConfig) (*execrpc.Client[model.Config, archiveplugin.Request, any, model.Receipt], error) {
	env := cfg.Options.Env
	var hasGoProxy bool
	for _, e := range env {
		if strings.HasPrefix(e, "GOPROXY=") {
			hasGoProxy = true
			break
		}
	}
	if !hasGoProxy {
		env = append(env, "GOPROXY="+cfg.GoSettings.GoProxy)
	}

	serverCfg := model.Config{
		Try: cfg.Try,
		ProjectInfo: model.ProjectInfo{
			Project: cfg.Project,
			Tag:     cfg.Tag,
		},
	}

	client, err := execrpc.StartClient(
		execrpc.ClientOptions[model.Config, archiveplugin.Request, any, model.Receipt]{
			Config: serverCfg,
			ClientRawOptions: execrpc.ClientRawOptions{
				Version: pluginProtocolVersion,
				Cmd:     cfg.GoSettings.GoExe,
				Args:    []string{"run", cfg.Options.Command},
				Dir:     cfg.Options.Dir,
				Env:     env,
				Timeout: 20 * time.Minute,
			},
			Codec: codecs.TOMLCodec{},
		},
	)
	if err != nil {
		return nil, err
	}

	go func() {
		for msg := range client.MessagesRaw() {
			statusCode := msg.Header.Status
			switch statusCode {
			case model.StatusInfoLog:
				cfg.Infol.Log(logg.String(string(msg.Body)))
			}
		}
	}()

	return client, nil
}
