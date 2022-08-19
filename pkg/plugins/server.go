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
	"fmt"

	"github.com/bep/execrpc"
)

// TODO(bep) move
const StatusInfoLog = 101

type Server[Q, R any] struct {
	*execrpc.Server[Q, R]
}

type Heartbeater[R any] interface {
	HeartbeatResponse() (R, bool)
}

// NewServer creates a new server which will call the given function with a request Q.
// The Dispatcher an be used for logging. Any errors needs to be sent in R.
func NewServer[Q Heartbeater[R], R any](call func(Dispatcher, Q) R) (*Server[Q, R], error) {
	rpcServer, err := execrpc.NewServer(
		execrpc.ServerOptions[Q, R]{
			Call: func(d execrpc.Dispatcher, req Q) R {
				if r, ok := req.HeartbeatResponse(); ok {
					return r
				}
				return call(dispatcherAdapter{d}, req)
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return &Server[Q, R]{
		Server: rpcServer,
	}, nil
}

type dispatcherAdapter struct {
	execRpcDispatcher execrpc.Dispatcher
}

func (d dispatcherAdapter) Infof(format string, args ...interface{}) {
	d.execRpcDispatcher.Send(execrpc.Message{
		Header: execrpc.Header{
			Status: StatusInfoLog,
		},
		Body: []byte(fmt.Sprintf(format, args...)),
	})
}

type Dispatcher interface {
	Infof(format string, args ...interface{})
}
