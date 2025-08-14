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
	goodBlob                     = "good blob"
	goodBlobDescriptor           = content.NewDescriptorFromBytes(ocispec.MediaTypeImageLayer, []byte(goodBlob))
	nonexistentBlobDescriptor    = ocispec.Descriptor{MediaType: ocispec.MediaTypeImageLayer, Digest: "sha256:9d16f5505246424aed7116cb21216704ba8c919997d0f1f37e154c11d509e1d2", Size: 123}
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
)

type TestGraphTarget struct {
	store *memory.Store
}

func (gt *TestGraphTarget) Exists(ctx context.Context, target ocispec.Descriptor) (bool, error) {
	return gt.store.Exists(ctx, target)
}

func (gt *TestGraphTarget) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	if target.Digest == badBlobSizeSmallerDescriptor.Digest {
		return io.NopCloser(bytes.NewReader([]byte(badBlobSizeSmaller))), nil
	}
	if target.Digest == badBlobSizeSmallerDescriptor.Digest {
		return io.NopCloser(bytes.NewReader([]byte(badBlobSizeSmaller))), nil
	}
	if target.Digest == badBlobSizeLargerDescriptor.Digest {
		return io.NopCloser(bytes.NewReader([]byte(badBlobSizeLarger))), nil
	}
	if target.Digest == badBlobDigestDescriptor.Digest {
		return io.NopCloser(bytes.NewReader([]byte(badBlobDigest))), nil
	}
	return gt.store.Fetch(ctx, target)
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
	return gt
}

func Test_checkBlobs(t *testing.T) {
	tests := []struct {
		name          string
		blobs         []ocispec.Descriptor
		expectedError error
	}{
		{
			name:          "a valid blob",
			blobs:         []ocispec.Descriptor{goodBlobDescriptor},
			expectedError: nil,
		},
		{
			name:          "a nonexistent blob",
			blobs:         []ocispec.Descriptor{nonexistentBlobDescriptor},
			expectedError: errdef.ErrNotFound,
		},
		{
			name:          "a blob with smaller size than the descriptor size",
			blobs:         []ocispec.Descriptor{badBlobSizeSmallerDescriptor},
			expectedError: io.ErrUnexpectedEOF,
		},
		{
			name:          "a blob with larger size than the descriptor size",
			blobs:         []ocispec.Descriptor{badBlobSizeLargerDescriptor},
			expectedError: content.ErrTrailingData,
		},
		{
			name:          "a blob with mismatched digest from the descriptor digest",
			blobs:         []ocispec.Descriptor{badBlobDigestDescriptor},
			expectedError: content.ErrMismatchedDigest,
		},
	}
	for _, tt := range tests {
		ctx := context.Background()
		gt := prepareTestGraphTarget(ctx)
		t.Run(tt.name, func(t *testing.T) {
			if err := checkBlobs(ctx, gt, tt.blobs); !errors.Is(err, tt.expectedError) {
				t.Errorf("checkBlobs() error = %v, wantErr %v", err, tt.expectedError)
			}
		})
	}
}
