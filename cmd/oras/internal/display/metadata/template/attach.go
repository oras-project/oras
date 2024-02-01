package template

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/model"
	"oras.land/oras/cmd/oras/internal/option"
)

type AttachHandler struct {
	template string
}

func NewAttachHandler(template string) metadata.AttachHandler {
	return &AttachHandler{template: template}
}

func (ah *AttachHandler) OnCompleted(opts *option.Target, root, subject ocispec.Descriptor) error {
	return parseAndWrite(model.NewPush(root, opts.Path), ah.template)
}
