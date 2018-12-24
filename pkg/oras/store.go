package oras

import (
	"bytes"
	"context"
	"io"

	"github.com/containerd/containerd/content"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ensure interface
var (
	_ content.Provider = &MemoryStore{}
)

// MemoryStore stores contents in the memory
type MemoryStore struct {
	content map[string][]byte
}

// NewMemoryStore creates a new memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		content: make(map[string][]byte),
	}
}

// Set adds the content to the store
func (s *MemoryStore) Set(desc ocispec.Descriptor, content []byte) {
	s.content[desc.Digest.String()] = content
}

// ReaderAt provides contents
func (s *MemoryStore) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (content.ReaderAt, error) {
	if content, ok := s.content[desc.Digest.String()]; ok {
		return newReaderAt(content), nil

	}
	return nil, ErrNotFound
}

type readerAt struct {
	io.ReaderAt
	size int64
}

func newReaderAt(content []byte) *readerAt {
	return &readerAt{
		ReaderAt: bytes.NewReader(content),
		size:     int64(len(content)),
	}
}

func (r *readerAt) Close() error {
	return nil
}

func (r *readerAt) Size() int64 {
	return r.size
}
