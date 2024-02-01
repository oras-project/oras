package json

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/model"
	"oras.land/oras/cmd/oras/internal/option"
)

type AttachHandler struct {
	path string
}

func NewAttachHandler() metadata.AttachHandler {
	return &AttachHandler{}
}

func (ah *AttachHandler) OnCopied(opts *option.Target) error {
	ah.path = opts.Path
	return nil
}

func (ah *AttachHandler) OnCompleted(root ocispec.Descriptor) error {
	return printJSON(model.NewPush(root, ah.path))
}
