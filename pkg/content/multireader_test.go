package content_test

import (
	"context"
	"io/ioutil"
	"testing"

	ctrcontent "github.com/containerd/containerd/content"
	"github.com/deislabs/oras/pkg/content"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
	testContentA     = []byte("Hello World!")
	testContentHashA = digest.FromBytes(testContentA)
	testContentB     = []byte("So long and thanks for all the fish!")
	testContentHashB = digest.FromBytes(testContentB)
	testDescriptorA  = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageConfig,
		Digest:    testContentHashA,
		Size:      int64(len(testContentA)),
	}
	testDescriptorB = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageConfig,
		Digest:    testContentHashB,
		Size:      int64(len(testContentB)),
	}
)

func TestMultiReader(t *testing.T) {
	mem1, mem2 := content.NewMemoryStore(), content.NewMemoryStore()
	mem1.Add("a", ocispec.MediaTypeImageConfig, testContentA)
	mem2.Add("b", ocispec.MediaTypeImageConfig, testContentB)
	multiReader := content.MultiReader{}
	multiReader.AddStore(mem1, mem2)

	ctx := context.Background()
	contentA, err := multiReader.ReaderAt(ctx, testDescriptorA)
	if err != nil {
		t.Fatalf("failed to get a reader for descriptor A: %v", err)
	}
	outputA, err := ioutil.ReadAll(ctrcontent.NewReader(contentA))
	if err != nil {
		t.Fatalf("failed to read content for descriptor A: %v", err)
	}
	if string(outputA) != string(testContentA) {
		t.Errorf("mismatched content for A, actual '%s', expected '%s'", outputA, testContentA)
	}

	contentB, err := multiReader.ReaderAt(ctx, testDescriptorB)
	if err != nil {
		t.Fatalf("failed to get a reader for descriptor B: %v", err)
	}
	outputB, err := ioutil.ReadAll(ctrcontent.NewReader(contentB))
	if err != nil {
		t.Fatalf("failed to read content for descriptor B: %v", err)
	}
	if string(outputB) != string(testContentB) {
		t.Errorf("mismatched content for B, actual '%s', expected '%s'", outputB, testContentB)
	}
}
