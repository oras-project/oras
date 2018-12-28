package oras

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"sync"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/remotes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

// ensure interface
var (
	_ content.Provider = &MemoryStore{}
)

// MemoryStore stores contents in the memory
type MemoryStore struct {
	content *sync.Map
}

// NewMemoryStore creates a new memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		content: &sync.Map{},
	}
}

// FetchHandler returnes a handler that will fetch all content into the memory store
// discovered in a call to Dispath.
// Use with ChildrenHandler to do a full recurisive fetch.
func (s *MemoryStore) FetchHandler(fetcher remotes.Fetcher) images.HandlerFunc {
	return func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		ctx = log.WithLogger(ctx, log.G(ctx).WithFields(logrus.Fields{
			"digest":    desc.Digest,
			"mediatype": desc.MediaType,
			"size":      desc.Size,
		}))

		log.G(ctx).Debug("fetch")
		rc, err := fetcher.Fetch(ctx, desc)
		if err != nil {
			return nil, err
		}
		defer rc.Close()

		content, err := ioutil.ReadAll(rc)
		if err != nil {
			return nil, err
		}
		s.Set(desc, content)
		return nil, nil
	}
}

// Set adds the content to the store
func (s *MemoryStore) Set(desc ocispec.Descriptor, content []byte) {
	s.content.Store(desc.Digest, content)
}

// Get finds the content from the store
func (s *MemoryStore) Get(desc ocispec.Descriptor) ([]byte, bool) {
	value, ok := s.content.Load(desc.Digest)
	if !ok {
		return nil, false
	}
	content, ok := value.([]byte)
	return content, ok
}

// ReaderAt provides contents
func (s *MemoryStore) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (content.ReaderAt, error) {
	if content, ok := s.Get(desc); ok {
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
