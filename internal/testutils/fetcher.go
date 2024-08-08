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

package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"io"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras/internal/docker"
)

// ErrorFetcher implements content.Fetcher.
type ErrorFetcher struct {
	ExpectedError error
}

// NewErrorFetcher create and error fetcher
func NewErrorFetcher() *ErrorFetcher {
	return &ErrorFetcher{
		ExpectedError: fmt.Errorf("expected error"),
	}
}

// Fetch returns an error.
func (f *ErrorFetcher) Fetch(context.Context, ocispec.Descriptor) (io.ReadCloser, error) {
	return nil, f.ExpectedError
}

type MockFetcher struct {
	store       *memory.Store
	Fetcher     content.Fetcher
	Subject     ocispec.Descriptor
	Config      ocispec.Descriptor
	OciImage    ocispec.Descriptor
	DockerImage ocispec.Descriptor
	Index       ocispec.Descriptor
}

// NewMockFetcher creates a MockFetcher and populates it.
func NewMockFetcher() (mockFetcher MockFetcher) {
	mockFetcher = MockFetcher{store: memory.New()}
	mockFetcher.Subject = mockFetcher.PushBlob(ocispec.MediaTypeImageLayer, []byte("blob"))
	imageType := "test.image"
	mockFetcher.Config = mockFetcher.PushBlob(imageType, []byte("config content"))
	mockFetcher.OciImage = mockFetcher.PushOCIImage(&mockFetcher.Subject, mockFetcher.Config)
	mockFetcher.OciImage.Annotations = map[string]string{ocispec.AnnotationTitle: "oci-image"}
	mockFetcher.DockerImage = mockFetcher.PushDockerImage(&mockFetcher.Subject, mockFetcher.Config)
	mockFetcher.Index = mockFetcher.PushIndex(mockFetcher.Subject)
	mockFetcher.Fetcher = mockFetcher.store
	return mockFetcher
}

// PushBlob pushes a blob to the memory store.
func (mf *MockFetcher) PushBlob(mediaType string, blob []byte) ocispec.Descriptor {
	desc := ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    digest.FromBytes(blob),
		Size:      int64(len(blob)),
	}
	if err := mf.store.Push(context.Background(), desc, bytes.NewReader(blob)); err != nil {
		panic(err)
	}
	return desc
}

func (mf *MockFetcher) pushImage(subject *ocispec.Descriptor, mediaType string, config ocispec.Descriptor, layers ...ocispec.Descriptor) ocispec.Descriptor {
	manifest := ocispec.Manifest{
		MediaType: mediaType,
		Subject:   subject,
		Config:    config,
		Layers:    layers,
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		panic(err)
	}
	return mf.PushBlob(mediaType, manifestJSON)
}

// PushOCIImage pushes the given subject, config and layers as a OCI image.
func (mf *MockFetcher) PushOCIImage(subject *ocispec.Descriptor, config ocispec.Descriptor, layers ...ocispec.Descriptor) ocispec.Descriptor {
	return mf.pushImage(subject, ocispec.MediaTypeImageManifest, config, layers...)
}

// PushDockerImage pushes the given subject, config and layers as a Docker image.
func (mf *MockFetcher) PushDockerImage(subject *ocispec.Descriptor, config ocispec.Descriptor, layers ...ocispec.Descriptor) ocispec.Descriptor {
	return mf.pushImage(subject, docker.MediaTypeManifest, config, layers...)
}

// PushIndex pushes the manifests as an index.
func (mf *MockFetcher) PushIndex(manifests ...ocispec.Descriptor) ocispec.Descriptor {
	index := ocispec.Index{
		Manifests: manifests,
	}
	indexJSON, err := json.Marshal(index)
	if err != nil {
		panic(err)
	}
	return mf.PushBlob(ocispec.MediaTypeImageIndex, indexJSON)
}
