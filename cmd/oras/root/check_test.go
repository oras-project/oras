/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package root

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/errdef"
)

var (
	// blob test data
	goodBlob                     = "good blob"
	goodBlobDescriptor           = content.NewDescriptorFromBytes(ocispec.MediaTypeImageLayer, []byte(goodBlob))
	badBlobSizeSmaller           = "bad blob size smaller than the descriptor size"
	badBlobSizeSmallerDescriptor = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageLayer,
		Digest:    digest.FromBytes([]byte(badBlobSizeSmaller)),
		Size:      int64(len(badBlobSizeSmaller)) + 1,
	}
	badBlobSizeLarger           = "bad blob size larger than the descriptor size"
	badBlobSizeLargerDescriptor = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageLayer,
		Digest:    digest.FromBytes([]byte(badBlobSizeLarger)),
		Size:      int64(len(badBlobSizeLarger)) - 1,
	}
	badBlobDigest           = "bad blob digest"
	badBlobDigestDescriptor = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageLayer,
		Digest:    "sha256:fd6ed2f36b5465244d5dc86cb4e7df0ab8a9d24adc57825099f522fe009a22bb",
		Size:      int64(len(badBlobDigest)),
	}
	nonexistentBlobDescriptor = ocispec.Descriptor{MediaType: ocispec.MediaTypeImageLayer, Digest: "sha256:9d16f5505246424aed7116cb21216704ba8c919997d0f1f37e154c11d509e1d2", Size: 123}

	// manifest test data
	goodManifest                     = `{"mediaType":"application/vnd.oci.image.manifest.v1+json"}`
	goodManifestDescriptor           = content.NewDescriptorFromBytes(ocispec.MediaTypeImageManifest, []byte(goodManifest))
	badManifestSizeSmaller           = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"}`
	badManifestSizeSmallerDescriptor = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    digest.FromBytes([]byte(badManifestSizeSmaller)),
		Size:      int64(len(badManifestSizeSmaller)) + 1,
	}
	badManifestSizeLarger           = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"},"config":{}`
	badManifestSizeLargerDescriptor = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    digest.FromBytes([]byte(badManifestSizeLarger)),
		Size:      int64(len(badManifestSizeLarger)) - 1,
	}
	badManifestDigest           = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"},"config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"sha256:fe9dbc99451d0517d65e048c309f0b5afb2cc513b7a3d456b6cc29fe641386c5","size":53}`
	badManifestDigestDescriptor = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    "sha256:fe9dbc99451d0517d65e048c309f0b5afb2cc513b7a3d456b6cc29fe641386c5",
		Size:      int64(len(badManifestDigest)),
	}
	badManifestMismatchedMediaType           = `{"mediaType":"application/vnd.docker.distribution.manifest.list.v2+json"}`
	badManifestMismatchedMediaTypeDescriptor = content.NewDescriptorFromBytes(ocispec.MediaTypeImageManifest, []byte(badManifestMismatchedMediaType))
	nonexistentManifestDescriptor            = ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: "sha256:9d16f5505246424aed7116cb21216704ba8c919997d0f1f37e154c11d509e1d2", Size: 123}
)

type TestGraphTarget struct {
	store *memory.Store
}

func (gt *TestGraphTarget) Exists(ctx context.Context, target ocispec.Descriptor) (bool, error) {
	return gt.store.Exists(ctx, target)
}

func (gt *TestGraphTarget) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	switch target.Digest {
	case badBlobSizeSmallerDescriptor.Digest:
		return io.NopCloser(bytes.NewReader([]byte(badBlobSizeSmaller))), nil
	case badBlobSizeLargerDescriptor.Digest:
		return io.NopCloser(bytes.NewReader([]byte(badBlobSizeLarger))), nil
	case badBlobDigestDescriptor.Digest:
		return io.NopCloser(bytes.NewReader([]byte(badBlobDigest))), nil
	case badManifestSizeSmallerDescriptor.Digest:
		return io.NopCloser(bytes.NewReader([]byte(badManifestSizeSmaller))), nil
	case badManifestSizeLargerDescriptor.Digest:
		return io.NopCloser(bytes.NewReader([]byte(badManifestSizeLarger))), nil
	case badManifestDigestDescriptor.Digest:
		return io.NopCloser(bytes.NewReader([]byte(badManifestDigest))), nil
	default:
		return gt.store.Fetch(ctx, target)
	}
}

func (gt *TestGraphTarget) Predecessors(ctx context.Context, node ocispec.Descriptor) ([]ocispec.Descriptor, error) {
	return gt.store.Predecessors(ctx, node)
}

func (gt *TestGraphTarget) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	return gt.store.Push(ctx, expected, content)
}

func (gt *TestGraphTarget) Resolve(ctx context.Context, reference string) (ocispec.Descriptor, error) {
	return gt.store.Resolve(ctx, reference)
}

func (gt *TestGraphTarget) Tag(ctx context.Context, desc ocispec.Descriptor, reference string) error {
	return gt.store.Tag(ctx, desc, reference)
}

func prepareTestGraphTarget(ctx context.Context) *TestGraphTarget {
	gt := &TestGraphTarget{
		store: memory.New(),
	}
	gt.Push(ctx, goodBlobDescriptor, bytes.NewReader([]byte(goodBlob)))
	gt.Push(ctx, goodManifestDescriptor, bytes.NewReader([]byte(goodManifest)))
	gt.Push(ctx, badManifestMismatchedMediaTypeDescriptor, bytes.NewReader([]byte(badManifestMismatchedMediaType)))
	return gt
}

func Test_checkBlobs(t *testing.T) {
	tests := []struct {
		name          string
		blob          ocispec.Descriptor
		expectedError error
	}{
		{
			name:          "a valid blob",
			blob:          goodBlobDescriptor,
			expectedError: nil,
		},
		{
			name:          "a blob with smaller size than the descriptor size",
			blob:          badBlobSizeSmallerDescriptor,
			expectedError: io.ErrUnexpectedEOF,
		},
		{
			name:          "a blob with larger size than the descriptor size",
			blob:          badBlobSizeLargerDescriptor,
			expectedError: content.ErrTrailingData,
		},
		{
			name:          "a blob with mismatched digest from the descriptor digest",
			blob:          badBlobDigestDescriptor,
			expectedError: content.ErrMismatchedDigest,
		},
		{
			name:          "a nonexistent blob",
			blob:          nonexistentBlobDescriptor,
			expectedError: errdef.ErrNotFound,
		},
	}
	ctx := context.Background()
	gt := prepareTestGraphTarget(ctx)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkBlob(ctx, gt, tt.blob); !errors.Is(err, tt.expectedError) {
				t.Errorf("checkBlob() error = %v, wantErr %v", err, tt.expectedError)
			}
		})
	}
}

func Test_checkManifest(t *testing.T) {
	tests := []struct {
		name          string
		manifest      ocispec.Descriptor
		expectedError error
	}{
		{
			name:          "a valid manifest",
			manifest:      goodManifestDescriptor,
			expectedError: nil,
		},
		{
			name:          "a manifest with smaller size than the descriptor size",
			manifest:      badManifestSizeSmallerDescriptor,
			expectedError: io.ErrUnexpectedEOF,
		},
		{
			name:          "a manifest with larger size than the descriptor size",
			manifest:      badManifestSizeLargerDescriptor,
			expectedError: content.ErrTrailingData,
		},
		{
			name:          "a manifest with mismatched digest from the descriptor digest",
			manifest:      badManifestDigestDescriptor,
			expectedError: content.ErrMismatchedDigest,
		},
		{
			name:          "a manifest with mismatched media type from the descriptor media type",
			manifest:      badManifestMismatchedMediaTypeDescriptor,
			expectedError: errMismatchedMediaType,
		},
		{
			name:          "a nonexistent manifest",
			manifest:      nonexistentManifestDescriptor,
			expectedError: errdef.ErrNotFound,
		},
	}
	ctx := context.Background()
	gt := prepareTestGraphTarget(ctx)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := checkManifest(ctx, gt, tt.manifest); !errors.Is(err, tt.expectedError) {
				t.Errorf("checkManifest() error = %v, wantErr %v", err, tt.expectedError)
			}
		})
	}
}
