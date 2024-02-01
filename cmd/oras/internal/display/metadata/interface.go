package metadata

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/option"
)

type PushHandler interface {
	OnCopied(opts *option.Target) error
	OnTagged(reference string) error
	OnCompleted(root ocispec.Descriptor) error
}

type AttachHandler interface {
	OnCompleted(opts *option.Target, root, subject ocispec.Descriptor) error
}
