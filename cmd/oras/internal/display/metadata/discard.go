package metadata

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/option"
)

type DiscardHandler struct{}

func NewDiscardHandler() DiscardHandler {
	return DiscardHandler{}
}

func (DiscardHandler) OnCopied(opts *option.Target) error {
	return nil
}

func (DiscardHandler) OnTagged(reference string) error {
	return nil
}

func (DiscardHandler) OnCompleted(root ocispec.Descriptor) error {
	return nil
}
