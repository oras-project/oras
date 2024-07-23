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

package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/opencontainers/go-digest"
	"oras.land/oras-go/v2/content/memory"
	"reflect"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/internal/docker"
)

type contentFetcher struct {
	content.Fetcher
}

func newTestFetcher(t *testing.T) (subject, config, ociImage, dockerImage, index ocispec.Descriptor, fetcher content.Fetcher) {
	var blobs [][]byte
	ctx := context.Background()
	memoryStorage := memory.New()
	appendBlob := func(mediaType string, blob []byte) ocispec.Descriptor {
		blobs = append(blobs, blob)
		desc := ocispec.Descriptor{
			MediaType: mediaType,
			Digest:    digest.FromBytes(blob),
			Size:      int64(len(blob)),
		}
		if err := memoryStorage.Push(ctx, desc, bytes.NewReader(blob)); err != nil {
			t.Errorf("Error pushing %v\n", err)
		}
		return desc
	}
	generateImage := func(subject *ocispec.Descriptor, mediaType string, config ocispec.Descriptor, layers ...ocispec.Descriptor) ocispec.Descriptor {
		manifest := ocispec.Manifest{
			MediaType: mediaType,
			Subject:   subject,
			Config:    config,
			Layers:    layers,
		}
		manifestJSON, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		return appendBlob(mediaType, manifestJSON)
	}
	generateIndex := func(manifests ...ocispec.Descriptor) ocispec.Descriptor {
		index := ocispec.Index{
			Manifests: manifests,
		}
		indexJSON, err := json.Marshal(index)
		if err != nil {
			t.Fatal(err)
		}
		return appendBlob(ocispec.MediaTypeImageIndex, indexJSON)
	}

	subject = appendBlob(ocispec.MediaTypeImageLayer, []byte("blob"))
	imageType := "test.image"
	config = appendBlob(imageType, []byte("config content"))
	ociImage = generateImage(&subject, ocispec.MediaTypeImageManifest, config)
	dockerImage = generateImage(&subject, docker.MediaTypeManifest, config)
	index = generateIndex(subject)

	return subject, config, ociImage, dockerImage, index, &contentFetcher{Fetcher: memoryStorage}
}

func TestSuccessors(t *testing.T) {
	subject, config, ociImage, dockerImage, index, fetcher := newTestFetcher(t)
	ctx := context.Background()
	type args struct {
		ctx     context.Context
		fetcher content.Fetcher
		node    ocispec.Descriptor
	}
	tests := []struct {
		name        string
		args        args
		wantNodes   []ocispec.Descriptor
		wantSubject *ocispec.Descriptor
		wantConfig  *ocispec.Descriptor
		wantErr     bool
	}{
		{"should failed to get non-existent OCI image", args{ctx, fetcher, ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest}}, nil, nil, nil, true},
		{"should failed to get non-existent docker image", args{ctx, fetcher, ocispec.Descriptor{MediaType: docker.MediaTypeManifest}}, nil, nil, nil, true},
		{"should get success of a docker image", args{ctx, fetcher, dockerImage}, nil, &subject, &config, false},
		{"should get success of an OCI image", args{ctx, fetcher, ociImage}, nil, &subject, &config, false},
		{"should get success of an index", args{ctx, fetcher, index}, []ocispec.Descriptor{subject}, nil, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNodes, gotSubject, gotConfig, err := Successors(tt.args.ctx, tt.args.fetcher, tt.args.node)
			if (err != nil) != tt.wantErr {
				t.Errorf("Successors() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotNodes, tt.wantNodes) {
				t.Errorf("Successors() gotNodes = %v, want %v", gotNodes, tt.wantNodes)
			}
			if !reflect.DeepEqual(gotSubject, tt.wantSubject) {
				t.Errorf("Successors() gotSubject = %v, want %v", gotSubject, tt.wantSubject)
			}
			if !reflect.DeepEqual(gotConfig, tt.wantConfig) {
				t.Errorf("Successors() gotConfig = %v, want %v", gotConfig, tt.wantConfig)
			}
		})
	}
}
