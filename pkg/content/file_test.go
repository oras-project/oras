package content_test

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	ctrcontent "github.com/containerd/containerd/content"
	"github.com/deislabs/oras/pkg/content"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestFileStoreNoName(t *testing.T) {
	testContent := []byte("Hello World!")
	descriptor := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageConfig,
		Digest:    digest.FromBytes(testContent),
		Size:      int64(len(testContent)),
		// do NOT add the AnnotationTitle here; it is the essence of the test
	}

	tests := []struct {
		opts []content.WriterOpt
		err  error
	}{
		{nil, content.ErrNoName},
		{[]content.WriterOpt{content.WithIgnoreNoName()}, nil},
	}
	for _, tt := range tests {
		rootPath, err := ioutil.TempDir("", "oras_filestore_test")
		if err != nil {
			t.Fatalf("error creating tempdir: %v", err)
		}
		defer os.RemoveAll(rootPath)
		fileStore := content.NewFileStore(rootPath, tt.opts...)
		ctx := context.Background()
		refOpt := ctrcontent.WithDescriptor(descriptor)
		if _, err := fileStore.Writer(ctx, refOpt); err != tt.err {
			t.Errorf("mismatched error, actual '%v', expected '%v'", err, tt.err)
		}

	}
}
