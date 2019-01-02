package content

import (
	"context"
	"os"
	"sync"

	"github.com/containerd/containerd/content"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ensure interface
var (
	_ content.Provider = &FileProvider{}
)

// FileProvider provides content from the file system
type FileProvider struct {
	descriptor *sync.Map // map[string]ocispec.Descriptor
}

// NewFileProvider creats a new file provider
func NewFileProvider() *FileProvider {
	return nil
}

// Add adds a file reference
func (p *FileProvider) Add(filename, mediaType string) error {
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return err
	}
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	digest, err := digest.FromReader(file)
	if err != nil {
		return err
	}

	desc := ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    digest,
		Size:      fileInfo.Size(),
		Annotations: map[string]string{
			ocispec.AnnotationTitle: filename,
		},
	}

	p.descriptor.Store(desc.Digest, desc)
	return nil
}

// ReaderAt provides contents
func (p *FileProvider) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (content.ReaderAt, error) {
	value, ok := p.descriptor.Load(desc.Digest)
	if !ok {
		return nil, ErrNotFound
	}
	desc = value.(ocispec.Descriptor)
	filename, ok := desc.Annotations[ocispec.AnnotationTitle]
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	return sizeReaderAt{
		readAtCloser: file,
		size:         desc.Size,
	}, nil
}
