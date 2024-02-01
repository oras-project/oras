package json

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/model"
	"oras.land/oras/cmd/oras/internal/option"
)

type PushHandler struct {
	path string
}

func NewPushHandler() metadata.PushHandler {
	return &PushHandler{}
}

func (ph *PushHandler) OnCopied(opts *option.Target) error {
	ph.path = opts.Path
	return nil
}

func (ph *PushHandler) OnTagged(reference string) error {
	return nil
}

func (ph *PushHandler) OnCompleted(root ocispec.Descriptor) error {
	return printJSON(model.NewPush(root, ph.path))
}
