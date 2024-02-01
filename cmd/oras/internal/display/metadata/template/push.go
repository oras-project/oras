package template

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/model"
	"oras.land/oras/cmd/oras/internal/option"
)

type PushHandler struct {
	template string
	path     string
}

func NewPushHandler(template string) metadata.PushHandler {
	return &PushHandler{template: template}
}

func (ph *PushHandler) OnCopied(opts *option.Target) error {
	ph.path = opts.Path
	return nil
}

func (ph *PushHandler) OnTagged(reference string) error {
	return nil
}

func (ph *PushHandler) OnCompleted(root ocispec.Descriptor) error {
	return parseAndWrite(model.NewPush(root, ph.path), ph.template)
}
