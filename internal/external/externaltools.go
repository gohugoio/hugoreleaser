package external

import (
	"github.com/bep/execrpc"
	"github.com/bep/hugoreleaser/internal/external/messages"
	"github.com/bep/hugoreleaser/pkg/plugins"
)

type GoRun struct {
}

func BuildArchive(req messages.ArchiveRequest) (messages.ArchiveResponse, error) {
	rpcclient, err := execrpc.StartClient(
		execrpc.ClientOptions{
			Version: 1,
			Cmd:     "go",
			Args:    []string{"run", "/Users/bep/dev/go/bep/hugoreleaser/examples/plugins/archives/tar/main.go"},
		},
	)

	client := tclient[messages.ArchiveRequest, messages.ArchiveResponse]{
		Client: rpcclient,
		codec:  plugins.TOMLCodec[messages.ArchiveResponse, messages.ArchiveRequest]{},
	}

	if err != nil {
		return messages.ArchiveResponse{}, err
	}

	// TODO(bep) registry/startup.
	defer client.Close()

	return client.Call(req)

}

type tclient[T, S any] struct {
	*execrpc.Client
	codec plugins.TOMLCodec[S, T]
}

func (c tclient[T, S]) Call(req T) (S, error) {
	var s S
	body, err := c.codec.Encode(req)
	if err != nil {
		return s, err
	}

	resp, err := c.Client.Execute(body)
	if err != nil {
		return s, err
	}

	if err := c.codec.Decode(resp.Body, s); err != nil {
		return s, err
	}

	return s, nil

}
