package text

import (
	"fmt"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/option"
)

type AttachHandler struct{}

func NewAttachHandler() metadata.AttachHandler {
	return AttachHandler{}
}

func (AttachHandler) OnCopied(opts *option.Target) error {
	_, err := fmt.Println("Attached to", opts.AnnotatedReference())
	return err
}

func (AttachHandler) OnCompleted(root ocispec.Descriptor) error {
	_, err := fmt.Println("Digest:", root.Digest)
	return err
}
