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

// NewServer creates a new server which will call the given function with a request Q.
// The Dispatcher an be used for logging. Any errors needs to be sent in R.
func NewServer[Q, R any](call func(Dispatcher, Q) R) (*Server[Q, R], error) {
	rpcServer, err := execrpc.NewServer(
		execrpc.ServerOptions[Q, R]{
			Call: func(d execrpc.Dispatcher, req Q) R {
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
