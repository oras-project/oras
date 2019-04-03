package oras

import ocispec "github.com/opencontainers/image-spec/specs-go/v1"

type pushOpts struct {
	config *ocispec.Descriptor
}

// PushOpt allows callers to set options on the oras push
type PushOpt func(o *pushOpts) error

// WithConfig overrides the config
func WithConfig(config ocispec.Descriptor) PushOpt {
	return func(o *pushOpts) error {
		o.config = &config
		return nil
	}
}
