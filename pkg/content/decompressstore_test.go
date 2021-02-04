package content_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"testing"

	ctrcontent "github.com/containerd/containerd/content"
	"github.com/deislabs/oras/pkg/content"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestDecompressStore(t *testing.T) {
	rawContent := []byte("Hello World!")
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(rawContent); err != nil {
		t.Fatalf("unable to create gzip content for testing: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("unable to close gzip writer creating content for testing: %v", err)
	}
	gzipContent := buf.Bytes()
	gzipContentHash := digest.FromBytes(gzipContent)
	gzipDescriptor := ocispec.Descriptor{
		MediaType: fmt.Sprintf("%s+gzip", ocispec.MediaTypeImageConfig),
		Digest:    gzipContentHash,
		Size:      int64(len(gzipContent)),
	}

	memStore := content.NewMemoryStore()
	decompressStore := content.NewDecompressStore(memStore, content.WithBlocksize(0))
	ctx := context.Background()
	decompressWriter, err := decompressStore.Writer(ctx, ctrcontent.WithDescriptor(gzipDescriptor))
	if err != nil {
		t.Fatalf("unable to get a decompress writer: %v", err)
	}
	n, err := decompressWriter.Write(gzipContent)
	if err != nil {
		t.Fatalf("failed to write to decompress writer: %v", err)
	}
	if n != len(gzipContent) {
		t.Fatalf("wrote %d instead of expected %d bytes", n, len(gzipContent))
	}
	if err := decompressWriter.Commit(ctx, int64(len(gzipContent)), gzipContentHash); err != nil {
		t.Fatalf("unexpected error committing decompress writer: %v", err)
	}

	// and now we should be able to get the decompressed data from the memory store
	_, b, found := memStore.Get(gzipDescriptor)
	if !found {
		t.Fatalf("failed to get data from underlying memory store: %v", err)
	}
	if string(b) != string(rawContent) {
		t.Errorf("mismatched data in underlying memory store, actual '%s', expected '%s'", b, rawContent)
	}
}
