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
	"errors"
	"reflect"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
)

type errLister struct {
	oras.ReadOnlyGraphTarget
}

func (e *errLister) Referrers(ctx context.Context, desc ocispec.Descriptor, artifactType string, fn func(referrers []ocispec.Descriptor) error) error {
	return errors.New("")
}

type refLister struct {
	referrers []ocispec.Descriptor
	oras.ReadOnlyGraphTarget
}

func (m *refLister) Referrers(ctx context.Context, desc ocispec.Descriptor, artifactType string, fn func(referrers []ocispec.Descriptor) error) error {
	return fn(m.referrers)
}

type predecessorFinder struct {
	*memory.Store
}

func TestReferrers(t *testing.T) {
	ctx := context.Background()
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
	generateImage := func(subject *ocispec.Descriptor, annotations map[string]string, config ocispec.Descriptor, layers ...ocispec.Descriptor) {
		manifest := ocispec.Manifest{
			Subject:     subject,
			Config:      config,
			Layers:      layers,
			Annotations: annotations,
		}
		manifestJSON, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		appendBlob(ocispec.MediaTypeImageManifest, manifestJSON)
	}
	generateArtifact := func(artifactType string, subject *ocispec.Descriptor, annotations map[string]string, blobs ...ocispec.Descriptor) {
		manifest := ocispec.Artifact{
			Subject:      subject,
			Blobs:        blobs,
			Annotations:  annotations,
			ArtifactType: artifactType,
		}
		manifestJSON, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		appendBlob(ocispec.MediaTypeArtifactManifest, manifestJSON)
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
		imgConfig
		image
		artifact
		index
	)
	anno := map[string]string{"test": "foo"}
	appendBlob(ocispec.MediaTypeArtifactManifest, []byte("subject content"))
	imageType := "test.image"
	appendBlob(imageType, []byte("config content"))
	generateImage(&descs[subject], anno, descs[imgConfig])
	imageDesc := descs[image]
	imageDesc.Annotations = anno
	imageDesc.ArtifactType = imageType
	artifactType := "test.artifact"
	generateArtifact(artifactType, &descs[subject], anno)
	generateIndex(descs[subject])
	artifactDesc := descs[artifact]
	artifactDesc.Annotations = anno
	artifactDesc.ArtifactType = artifactType

	referrers := []ocispec.Descriptor{descs[image], descs[image]}
	memory := memory.New()
	for i := range descs {
		memory.Push(ctx, descs[i], bytes.NewReader(blobs[i]))
	}
	finder := &predecessorFinder{Store: memory}

	type args struct {
		ctx          context.Context
		target       oras.ReadOnlyGraphTarget
		desc         ocispec.Descriptor
		artifactType string
	}
	tests := []struct {
		name    string
		args    args
		want    []ocispec.Descriptor
		wantErr bool
	}{
		// TODO: Add test cases.
		{"should fail when a referrer lister failed to get referrers", args{ctx, &errLister{}, ocispec.Descriptor{}, ""}, nil, true},
		{"should return referrers when target is a referrer lister", args{ctx, &refLister{referrers: referrers}, ocispec.Descriptor{}, ""}, referrers, false},
		{"should return nil for index node", args{ctx, finder, descs[index], ""}, nil, false},
		{"should return nil for config node", args{ctx, finder, descs[imgConfig], ""}, nil, false},
		{"should find filtered image referrer", args{ctx, finder, descs[subject], imageType}, []ocispec.Descriptor{imageDesc}, false},
		{"should find filtered artifact referrer", args{ctx, finder, descs[subject], artifactType}, []ocispec.Descriptor{artifactDesc}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Referrers(tt.args.ctx, tt.args.target, tt.args.desc, tt.args.artifactType)
			if (err != nil) != tt.wantErr {
				t.Errorf("Referrers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Referrers() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("should find referrers in predecessors", func(t *testing.T) {
		want1 := []ocispec.Descriptor{artifactDesc, imageDesc}
		want2 := []ocispec.Descriptor{imageDesc, artifactDesc}
		got, err := Referrers(ctx, finder, descs[subject], "")
		if err != nil {
			t.Errorf("Referrers() error = %v", err)
			return
		}
		if !reflect.DeepEqual(got, want1) && !reflect.DeepEqual(got, want2) {
			t.Errorf("Referrers() = %v, want %v", got, want1)
		}
	})
}
