package oras

import (
	"context"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	artifact "github.com/deislabs/oras/pkg/artifact"
	digest "github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
)

// Push pushes files to the remote
func Push(ctx context.Context, resolver remotes.Resolver, ref string, provider content.Provider, descriptors []ocispec.Descriptor, opts ...PushOpt) (ocispec.Descriptor, error) {
	if resolver == nil {
		return ocispec.Descriptor{}, ErrResolverUndefined
	}
	opt := pushOptsDefaults()
	for _, o := range opts {
		if err := o(opt); err != nil {
			return ocispec.Descriptor{}, err
		}
	}
	if opt.validateName != nil {
		for _, desc := range descriptors {
			if err := opt.validateName(desc); err != nil {
				return ocispec.Descriptor{}, err
			}
		}
	}

	pusher, err := resolver.Pusher(ctx, ref)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	desc, store, err := pack(provider, descriptors, opt)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	var wrapper func(images.Handler) images.Handler
	if len(opt.baseHandlers) > 0 {
		wrapper = func(h images.Handler) images.Handler {
			return images.Handlers(append(opt.baseHandlers, h)...)
		}
	}

	if err := remotes.PushContent(ctx, pusher, desc, store, nil, nil, wrapper); err != nil {
		return ocispec.Descriptor{}, err
	}
	return desc, nil
}

func pack(provider content.Provider, descriptors []ocispec.Descriptor, opts *pushOpts) (ocispec.Descriptor, content.Store, error) {
	store := newHybridStoreFromProvider(provider, nil)
	if opts.manifest != nil {
		return *opts.manifest, store, nil
	}
	if descriptors == nil {
		descriptors = []ocispec.Descriptor{} // make it an empty array to prevent potential server-side bugs
	}

	// Config
	var config *ocispec.Descriptor
	if opts.config != nil {
		config = opts.config
	} else if opts.artifact == nil {
		configBytes := []byte("{}")
		config = &ocispec.Descriptor{
			MediaType: artifact.UnknownConfigMediaType,
			Digest:    digest.FromBytes(configBytes),
			Size:      int64(len(configBytes)),
		}
		store.Set(*config, configBytes)
	}
	if config != nil {
		if opts.configAnnotations != nil {
			config.Annotations = opts.configAnnotations
		}
		if opts.configMediaType != "" {
			config.MediaType = opts.configMediaType
		}
	}

	// Manifest
	var mediaType string
	var content interface{}
	if opts.artifact != nil {
		artifact := *opts.artifact
		artifact.Blobs = convertV1DescriptorsToV2(descriptors)
		artifact.Annotations = opts.manifestAnnotations
		mediaType = artifactspec.MediaTypeArtifactManifest
		content = artifact
	} else {
		mediaType = ocispec.MediaTypeImageManifest
		content = ocispec.Manifest{
			Versioned: specs.Versioned{
				SchemaVersion: 2, // historical value. does not pertain to OCI or docker version
			},
			Config:      *config,
			Layers:      descriptors,
			Annotations: opts.manifestAnnotations,
		}
	}
	desc, contentBytes, err := store.SetObject(mediaType, content)
	if err != nil {
		return ocispec.Descriptor{}, nil, err
	}
	if opts.manifestWriter != nil {
		if _, err := opts.manifestWriter.Write(contentBytes); err != nil {
			return ocispec.Descriptor{}, nil, err
		}
	}

	return desc, store, nil
}

func convertV1DescriptorsToV2(descs []ocispec.Descriptor) []artifactspec.Descriptor {
	results := make([]artifactspec.Descriptor, 0, len(descs))
	for _, desc := range descs {
		results = append(results, convertV1DescriptorToV2(desc))
	}
	return results
}

func convertV1DescriptorToV2(desc ocispec.Descriptor) artifactspec.Descriptor {
	return artifactspec.Descriptor{
		MediaType:   desc.MediaType,
		Digest:      desc.Digest,
		Size:        desc.Size,
		URLs:        desc.URLs,
		Annotations: desc.Annotations,
	}
}
