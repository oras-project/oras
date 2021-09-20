package images

import (
	"context"
	"encoding/json"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
)

// AppendArtifactsHandler will append artifacts desc to descs
func AppendArtifactsHandler(provider content.Provider) images.Handler {
	return images.HandlerFunc(func(ctx context.Context, desc v1.Descriptor) ([]v1.Descriptor, error) {
		descs := make([]v1.Descriptor, 0)

		switch desc.MediaType {

		case artifactspec.MediaTypeArtifactManifest:
			p, err := content.ReadBlob(ctx, provider, desc)
			if err != nil {
				return nil, err
			}

			artifact := &artifactspec.Manifest{}
			if err := json.Unmarshal(p, artifact); err != nil {
				return nil, err
			}

			appendDesc := func(artifacts ...artifactspec.Descriptor) {
				for _, desc := range artifacts {
					descs = append(descs, v1.Descriptor{
						MediaType:   desc.MediaType,
						Digest:      desc.Digest,
						Size:        desc.Size,
						URLs:        desc.URLs,
						Annotations: desc.Annotations,
					})
				}
			}

			appendDesc(artifact.Blobs...)
		}
		return descs, nil
	})
}
