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

// mock contains mocked components for unit testing
package mock

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry/remote"
)

type content struct {
	ocispec.Descriptor
	blob []byte
}

type repository struct {
	cas map[string]content
	remote.Repository
	isFetcher          bool
	isReferenceFetcher bool
	isResolver         bool
}

// WithFetch enables mocking for Fetch.
func (repo *repository) WithFetch() *repository {
	repo.isFetcher = true
	return repo
}

// WithFetchReference enables mocking for FetchReference.
func (repo *repository) WithFetchReference() *repository {
	repo.isReferenceFetcher = true
	return repo
}

// WithResolve enables mocking for Resolve.
func (repo *repository) WithResolve() *repository {
	repo.isResolver = true
	return repo
}

// New returns a new repository struct.
func New() *repository {
	return &repository{}
}

// Blob represents a content blob to mock in CAS.
type Blob struct {
	Content   string
	MediaType string
	Tag       string
}

// Remount remounts blobs to the CAS of the repo.
func (repo *repository) Remount(blobs []Blob) {
	repo.cas = make(map[string]content)
	for _, blob := range blobs {
		bytes := []byte(blob.Content)
		desc := ocispec.Descriptor{
			MediaType: blob.MediaType,
			Digest:    digest.FromBytes(bytes),
			Size:      int64(len(bytes)),
		}
		repo.cas[string(desc.Digest)] = content{desc, bytes}
		if blob.Tag != "" {
			repo.cas[blob.Tag] = content{desc, bytes}
		}
	}
}

var errNotImplemented = errors.New("not implemented")

// FetchReference mocks the fetching with a reference.
func (repo *repository) FetchReference(ctx context.Context, ref string) (ocispec.Descriptor, io.ReadCloser, error) {
	if repo.isReferenceFetcher {
		if c, ok := repo.cas[ref]; ok {
			return c.Descriptor, io.NopCloser(bytes.NewReader(c.blob)), nil
		}
		return ocispec.Descriptor{}, nil, fmt.Errorf("got unexpected reference %q", ref)
	}
	return ocispec.Descriptor{}, nil, errNotImplemented
}

// Fetch mocks fetching.
func (repo *repository) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	if repo.isFetcher {
		if r, ok := repo.cas[target.Digest.String()]; ok {
			return io.NopCloser(bytes.NewReader(r.blob)), nil
		}
		return nil, fmt.Errorf("unexpected descriptor %v", target)
	}
	return nil, errNotImplemented
}

// Resolve mocks resolving.
func (repo *repository) Resolve(ctx context.Context, reference string) (ocispec.Descriptor, error) {
	if repo.isResolver {
		if r, ok := repo.cas[reference]; ok {
			return r.Descriptor, nil
		}
		return ocispec.Descriptor{}, fmt.Errorf("unexpected reference %v", reference)
	}
	return ocispec.Descriptor{}, errNotImplemented
}
