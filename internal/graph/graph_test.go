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
	"reflect"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras/internal/docker"
)

type fetcher struct {
	content.Fetcher
}

func TestSuccessors(t *testing.T) {
	var blobs [][]byte
	var descs []ocispec.Descriptor
	appendBlob := func(mediaType string, blob []byte) {
		blobs = append(blobs, blob)
		descs = append(descs, ocispec.Descriptor{
			MediaType: mediaType,
			Digest:    digest.FromBytes(blob),
			Size:      int64(len(blob)),
		})
	}
	generateImage := func(subject *ocispec.Descriptor, mediaType string, config ocispec.Descriptor, layers ...ocispec.Descriptor) {
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
		appendBlob(mediaType, manifestJSON)
	}
	generateIndex := func(manifests ...ocispec.Descriptor) {
		index := ocispec.Index{
			Manifests: manifests,
		}
		manifestJSON, err := json.Marshal(index)
		if err != nil {
			t.Fatal(err)
		}
		appendBlob(ocispec.MediaTypeImageIndex, manifestJSON)
	}
	const (
		subject = iota
		config
		ociImage
		dockerImage
		index
	)
	appendBlob(ocispec.MediaTypeImageLayer, []byte("blob"))
	imageType := "test.image"
	appendBlob(imageType, []byte("config content"))
	generateImage(&descs[subject], ocispec.MediaTypeImageManifest, descs[config])
	generateImage(&descs[subject], docker.MediaTypeManifest, descs[config])
	generateIndex(descs[subject])
	memory := memory.New()
	ctx := context.Background()
	for i := range descs {
		if err := memory.Push(ctx, descs[i], bytes.NewReader(blobs[i])); err != nil {
			t.Errorf("Error pushing %v\n", err)
		}
	}
	fetcher := &fetcher{Fetcher: memory}

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
		{"should get success of a docker image", args{ctx, fetcher, descs[dockerImage]}, nil, &descs[subject], &descs[config], false},
		{"should get success of an OCI image", args{ctx, fetcher, descs[ociImage]}, nil, &descs[subject], &descs[config], false},
		{"should get success of an index", args{ctx, fetcher, descs[index]}, []ocispec.Descriptor{descs[subject]}, nil, nil, false},
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
