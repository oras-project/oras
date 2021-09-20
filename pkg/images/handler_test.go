package images

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/containerd/containerd/content"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
)

type (
	testContentProvider struct {
		content.Provider
	}
)

func (testContentProvider) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (content.ReaderAt, error) {
	return &testContentProvider{}, nil
}

func (testContentProvider) ReadAt(p []byte, off int64) (n int, err error) {
	manifest := testArtifactsManifest()

	b, err := json.Marshal(manifest)
	if err != nil {
		return 0, err
	}

	return copy(p, b), nil
}

func (testContentProvider) Close() error {
	return nil
}

func (testContentProvider) Size() int64 {
	manifest := testArtifactsManifest()

	b, err := json.Marshal(manifest)
	if err != nil {
		return -1
	}

	return int64(len(b))
}

func testArtifactsManifest() *artifactspec.Manifest {
	return &artifactspec.Manifest{
		Blobs: []artifactspec.Descriptor{
			{
				ArtifactType: "sbom/example",
				MediaType:    "application/json",
				Digest:       "test-sbom-example",
			},
			{
				ArtifactType: "signature/example",
				MediaType:    "application/json",
				Digest:       "test-signature-example",
			},
		},
	}
}

func TestAppendArtifactsHandler(t *testing.T) {
	handler := AppendArtifactsHandler(&testContentProvider{})

	subdescs, err := handler.Handle(context.Background(), ocispec.Descriptor{MediaType: artifactspec.MediaTypeArtifactManifest})
	if err != nil {
		t.Error(err)
	}

	if subdescs[0].MediaType != "application/json" {
		t.FailNow()
	}

	if subdescs[0].Digest != "test-sbom-example" {
		t.FailNow()
	}

	if subdescs[1].MediaType != "application/json" {
		t.FailNow()
	}

	if subdescs[1].Digest != "test-signature-example" {
		t.FailNow()
	}
}
