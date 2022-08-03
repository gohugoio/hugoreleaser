package main

import (
	"os"

	"github.com/bep/hugoreleaser/internal/external/messages"
	"github.com/bep/hugoreleaser/pkg/plugins"
)

func main() {
	// TODO(bep) move types out of internal if needed.
	callback := func(req messages.ArchiveRequest) (messages.ArchiveResponse, error) {
		return messages.ArchiveResponse{
			Ext: ".foo",
		}, nil
	}
	server := plugins.NewServer(callback)

	if err := server.Start(); err != nil {
		print("error: failed to start tar archive server:", err)
		os.Exit(1)
	}

	_ = server.Wait()
}
