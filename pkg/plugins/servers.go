package plugins

import (
	"bytes"
	"os"

	"github.com/bep/execrpc"
	"github.com/pelletier/go-toml/v2"
)

// TODO(bep) move these common types.
type Codec[T, S any] interface {
	Decode([]byte, T) error
	Encode(T) ([]byte, error)
}

type TOMLCodec[T, S any] struct{}

func (c TOMLCodec[T, S]) Decode(b []byte, v T) error {
	return toml.Unmarshal(b, &v)
}

func (c TOMLCodec[T, S]) Encode(v S) ([]byte, error) {
	var b bytes.Buffer
	enc := toml.NewEncoder(&b)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

type Server[T, S any] struct {
	server *execrpc.Server
}

func (s Server[T, S]) Start() error {
	return s.server.Start()
}

func (s Server[T, S]) Wait() error {
	return s.server.Wait()
}

func NewServer[T, S any](call func(T) (S, error)) Server[T, S] {
	codec := TOMLCodec[T, S]{}
	server := execrpc.NewServer(
		execrpc.ServerOptions{
			In:  os.Stdin,
			Out: os.Stdout,
			Call: func(message execrpc.Message) execrpc.Message {
				var req T
				if err := codec.Decode(message.Body, req); err != nil {
					panic("error: TODO(bep)")
					//message.Header.Status = 32 // TODO(bep) improve error handling.
					//return message
				}

				resp, err := call(req)
				if err != nil {
					panic("error: TODO(bep)")
				}

				body, err := codec.Encode(resp)
				if err != nil {
					panic("error: TODO(bep)")
				}

				return execrpc.Message{
					Header: message.Header,
					Body:   body,
				}
			},
		},
	)
	return Server[T, S]{

		server: server,
	}
}
