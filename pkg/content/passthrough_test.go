package content_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	ctrcontent "github.com/containerd/containerd/content"
	"github.com/deislabs/oras/pkg/content"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
	testRef             = "abc123"
	testContent         = []byte("Hello World!")
	testContentHash     = digest.FromBytes(testContent)
	appendText          = "1"
	modifiedContent     = fmt.Sprintf("%s%s", testContent, appendText)
	modifiedContentHash = digest.FromBytes([]byte(modifiedContent))
	testDescriptor      = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageConfig,
		Digest:    testContentHash,
		Size:      int64(len(testContent)),
		Annotations: map[string]string{
			ocispec.AnnotationTitle: testRef,
		},
	}
	modifiedDescriptor = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageConfig,
		Digest:    modifiedContentHash,
		Size:      int64(len(modifiedContent)),
		Annotations: map[string]string{
			ocispec.AnnotationTitle: testRef,
		},
	}
)

func TestPassthroughWriter(t *testing.T) {
	// simple pass through function that modifies the data just slightly
	f := func(r io.Reader, w io.Writer, done chan<- error) {
		var (
			err error
			n   int
		)
		for {
			b := make([]byte, 1024)
			n, err = r.Read(b)
			if err != nil && err != io.EOF {
				t.Fatalf("data read error: %v", err)
				break
			}
			l := n
			if n > len(b) {
				l = len(b)
			}

			// we change it just slightly
			b = b[:l]
			if l > 0 {
				b = append(b, []byte(appendText)...)
			}
			if _, err := w.Write(b); err != nil {
				t.Fatalf("error writing to underlying writer: %v", err)
				break
			}
			if err == io.EOF {
				break
			}
		}
		done <- err
	}

	tests := []struct {
		opts []content.WriterOpt
		hash digest.Digest
	}{
		{nil, testContentHash},
		{[]content.WriterOpt{content.WithInputHash(testContentHash), content.WithOutputHash(modifiedContentHash)}, testContentHash},
	}

	for _, tt := range tests {
		ctx := context.Background()
		mem := content.NewMemoryStore()
		memw, err := mem.Writer(ctx, ctrcontent.WithDescriptor(modifiedDescriptor))
		if err != nil {
			t.Fatalf("unexpected error getting the memory store writer: %v", err)
		}
		writer := content.NewPassthroughWriter(memw, f, tt.opts...)
		n, err := writer.Write(testContent)
		if err != nil {
			t.Fatalf("unexpected error on Write: %v", err)
		}
		if n != len(testContent) {
			t.Fatalf("wrote %d bytes instead of %d", n, len(testContent))
		}
		if err := writer.Commit(ctx, testDescriptor.Size, tt.hash); err != nil {
			t.Errorf("unexpected error on Commit: %v", err)
		}
		if digest := writer.Digest(); digest != tt.hash {
			t.Errorf("mismatched digest: actual %v, expected %v", digest, tt.hash)
		}

		// make sure the data is what we expected
		_, b, found := mem.Get(modifiedDescriptor)
		if !found {
			t.Fatalf("target descriptor not found in underlying memory store")
		}
		if len(b) != len(modifiedContent) {
			t.Errorf("unexpectedly got %d bytes instead of expected %d", len(b), len(modifiedContent))
		}
		if string(b) != modifiedContent {
			t.Errorf("mismatched content, expected '%s', got '%s'", modifiedContent, string(b))
		}
	}
}
