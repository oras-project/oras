package text

import (
	"fmt"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/option"
)

type PushHandler struct{}

func NewPushHandler() metadata.PushHandler {
	return PushHandler{}
}

func (PushHandler) OnCopied(opts *option.Target) error {
	_, err := fmt.Println("Pushed", opts.AnnotatedReference())
	return err
}

func (PushHandler) OnTagged(reference string) error {
	panic("not implemented")
}

func (PushHandler) OnCompleted(root ocispec.Descriptor) error {
	_, err := fmt.Println("Digest:", root.Digest)
	return err
}
