package content_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
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

func TestPassthroughMultiWriter(t *testing.T) {
	// pass through function that selects one of two outputs
	var (
		b1, b2       bytes.Buffer
		name1, name2 = "I am name 01", "I am name 02" // each of these is 12 bytes
		data1, data2 = make([]byte, 500), make([]byte, 500)
	)
	rand.Read(data1)
	rand.Read(data2)
	combined := append([]byte(name1), data1...)
	combined = append(combined, []byte(name2)...)
	combined = append(combined, data2...)
	f := func(r io.Reader, getwriter func(name string) io.Writer, done chan<- error) {
		var (
			err error
		)
		// test is done rather simply, with a single 1024 byte chunk, split into 2x512 data streams, each of which is
		// 12 bytes of name and 500 bytes of data
		b := make([]byte, 1024)
		_, err = r.Read(b)
		if err != nil && err != io.EOF {
			t.Fatalf("data read error: %v", err)
		}

		// get the names and data for each
		n1, n2 := string(b[0:12]), string(b[512+0:512+12])
		w1, w2 := getwriter(n1), getwriter(n2)
		if _, err := w1.Write(b[12:512]); err != nil {
			t.Fatalf("w1 write error: %v", err)
		}
		if _, err := w2.Write(b[512+12 : 1024]); err != nil {
			t.Fatalf("w2 write error: %v", err)
		}
		done <- err
	}

	var (
		opts = []content.WriterOpt{content.WithInputHash(testContentHash), content.WithOutputHash(modifiedContentHash)}
		hash = testContentHash
	)
	ctx := context.Background()
	writers := func(name string) (ctrcontent.Writer, error) {
		switch name {
		case name1:
			return content.NewIoContentWriter(&b1), nil
		case name2:
			return content.NewIoContentWriter(&b2), nil
		}
		return nil, fmt.Errorf("unknown name %s", name)
	}
	writer := content.NewPassthroughMultiWriter(writers, f, opts...)
	n, err := writer.Write(combined)
	if err != nil {
		t.Fatalf("unexpected error on Write: %v", err)
	}
	if n != len(combined) {
		t.Fatalf("wrote %d bytes instead of %d", n, len(combined))
	}
	if err := writer.Commit(ctx, testDescriptor.Size, hash); err != nil {
		t.Errorf("unexpected error on Commit: %v", err)
	}
	if digest := writer.Digest(); digest != hash {
		t.Errorf("mismatched digest: actual %v, expected %v", digest, hash)
	}

	// make sure the data is what we expected
	if !bytes.Equal(data1, b1.Bytes()) {
		t.Errorf("b1 data1 did not match")
	}
	if !bytes.Equal(data2, b2.Bytes()) {
		t.Errorf("b2 data2 did not match")
	}
}
