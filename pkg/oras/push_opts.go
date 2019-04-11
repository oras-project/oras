package oras

import ocispec "github.com/opencontainers/image-spec/specs-go/v1"

type pushOpts struct {
	config              *ocispec.Descriptor
	configAnnotations   map[string]string
	manifestAnnotations map[string]string
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

// WithConfigAnnotations overrides the config annotations
func WithConfigAnnotations(annotations map[string]string) PushOpt {
	return func(o *pushOpts) error {
		o.configAnnotations = annotations
		return nil
	}
}

// WithManifestAnnotations overrides the manifest annotations
func WithManifestAnnotations(annotations map[string]string) PushOpt {
	return func(o *pushOpts) error {
		o.manifestAnnotations = annotations
		return nil
	}
}
