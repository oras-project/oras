package docker

import (
	"context"
	"strings"
	"sync"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/remotes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"
)

// PushContent pushes content specified by the descriptor from the provider.
//
// Base handlers can be provided which will be called before any push specific
// handlers.
func PushContent(ctx context.Context, pusher remotes.Pusher, desc ocispec.Descriptor, store content.Store, limiter *semaphore.Weighted, platform platforms.MatchComparer, wrapper func(h images.Handler) images.Handler) error {

	var m sync.Mutex
	manifestStack := []ocispec.Descriptor{}

	filterHandler := images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		switch desc.MediaType {
		case images.MediaTypeDockerSchema2Manifest, ocispec.MediaTypeImageManifest,
			images.MediaTypeDockerSchema2ManifestList, ocispec.MediaTypeImageIndex,
			artifactspec.MediaTypeArtifactManifest:
			m.Lock()
			manifestStack = append(manifestStack, desc)
			m.Unlock()
			return nil, images.ErrStopHandler
		default:
			return nil, nil
		}
	})

	pushHandler := remotes.PushHandler(pusher, store)

	platformFilterhandler := images.FilterPlatforms(images.ChildrenHandler(store), platform)

	annotateHandler := annotateDistributionSourceHandler(platformFilterhandler, store)

	var handler images.Handler = images.Handlers(
		annotateHandler,
		filterHandler,
		pushHandler,
	)
	if wrapper != nil {
		handler = wrapper(handler)
	}

	if err := images.Dispatch(ctx, handler, limiter, desc); err != nil {
		return err
	}

	// Iterate in reverse order as seen, parent always uploaded after child
	for i := len(manifestStack) - 1; i >= 0; i-- {
		_, err := pushHandler(ctx, manifestStack[i])
		if err != nil {
			// TODO(estesp): until we have a more complete method for index push, we need to report
			// missing dependencies in an index/manifest list by sensing the "400 Bad Request"
			// as a marker for this problem
			switch manifestStack[i].MediaType {
			case ocispec.MediaTypeImageIndex, images.MediaTypeDockerSchema2ManifestList,
				artifactspec.MediaTypeArtifactManifest:
				if errors.Cause(err) != nil && strings.Contains(errors.Cause(err).Error(), "400 Bad Request") {
					return errors.Wrap(err, "artifact or manifest list/index references to blobs and/or manifests are missing in your target registry")
				}
			}
			return err
		}
	}

	return nil
}

// annotateDistributionSourceHandler add distribution source label into
// annotation of config or blob descriptor.
func annotateDistributionSourceHandler(f images.HandlerFunc, manager content.Manager) images.HandlerFunc {
	return func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		children, err := f(ctx, desc)
		if err != nil {
			return nil, err
		}

		// only add distribution source for the config or blob data descriptor
		switch desc.MediaType {
		case images.MediaTypeDockerSchema2Manifest, ocispec.MediaTypeImageManifest,
			images.MediaTypeDockerSchema2ManifestList, ocispec.MediaTypeImageIndex,
			artifactspec.MediaTypeArtifactManifest:
		default:
			return children, nil
		}

		for i := range children {
			child := children[i]

			info, err := manager.Info(ctx, child.Digest)
			if err != nil {
				return nil, err
			}

			for k, v := range info.Labels {
				if !strings.HasPrefix(k, "containerd.io/distribution.source.") {
					continue
				}

				if child.Annotations == nil {
					child.Annotations = map[string]string{}
				}
				child.Annotations[k] = v
			}

			children[i] = child
		}
		return children, nil
	}
}
