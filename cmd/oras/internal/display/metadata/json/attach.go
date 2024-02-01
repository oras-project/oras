package json

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/model"
	"oras.land/oras/cmd/oras/internal/option"
)

type AttachHandler struct{}

func NewAttachHandler() metadata.AttachHandler {
	return AttachHandler{}
}

func (AttachHandler) OnCompleted(opts *option.Target, root, subject ocispec.Descriptor) error {
	return printJSON(model.NewPush(root, opts.Path))
}
