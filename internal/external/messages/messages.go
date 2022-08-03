package messages

import "github.com/bep/hugoreleaser/internal/model"

// ArchiveRequest is what is sent to an external acrhive tool
type ArchiveRequest struct {
	// Information about the build to archive.
	model.BuildContext
	// Filename without extension
	BaseOutFilename string `toml:"base_out_filename"`
}

// ArchiveResponse is what is sent back from an external archive tool
type ArchiveResponse struct {
	// File extension, including the dot.
	Ext string `toml:"ext"`
}
