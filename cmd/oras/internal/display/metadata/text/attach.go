package text

import (
	"fmt"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/option"
)

type AttachHandler struct{}

func NewAttachHandler() metadata.AttachHandler {
	return AttachHandler{}
}

func (AttachHandler) OnCompleted(opts *option.Target, root, subject ocispec.Descriptor) error {
	digest := subject.Digest.String()
	if !strings.HasSuffix(opts.RawReference, digest) {
		opts.RawReference = fmt.Sprintf("%s@%s", opts.Path, subject.Digest)
	}
	_, err := fmt.Println("Attached to", opts.AnnotatedReference())
	if err != nil {
		return err
	}
	_, err = fmt.Println("Digest:", root.Digest)
	return err
}
