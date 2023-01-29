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
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
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
	predecessors []ocispec.Descriptor
	oras.ReadOnlyGraphTarget
}

func (m *predecessorFinder) Predecessors(ctx context.Context, node ocispec.Descriptor) ([]ocispec.Descriptor, error) {
	return m.predecessors, nil
}

func TestReferrers(t *testing.T) {
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
			Subject:     subject,
			Blobs:       blobs,
			Annotations: annotations,
		}
		manifestJSON, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		appendBlob(ocispec.MediaTypeImageManifest, manifestJSON)
	}
	generateIndex := func(manifests ...ocispec.Descriptor) {
		index := ocispec.Index{
			Manifests: manifests,
		}
		manifestJSON, err := json.Marshal(index)
		if err != nil {
			t.Fatal(err)
		}
		appendBlob(ocispec.MediaTypeImageManifest, manifestJSON)
	}
	const (
		subject = iota
		imgConfig
		imgReferrer
		artifactReferrer
		index
	)
	anno := map[string]string{"test": "foo"}
	appendBlob(ocispec.MediaTypeArtifactManifest, []byte("subject content"))
	imageArtifactType := "test.image"
	appendBlob(imageArtifactType, []byte("config content"))
	generateImage(&descs[subject], anno, descs[imgConfig])
	artifactType := "test.artifact"
	generateArtifact(artifactType, &descs[subject], anno)
	generateIndex(descs[subject])

	referrers := []ocispec.Descriptor{descs[imgReferrer], descs[imgReferrer]}
	predecessors := []ocispec.Descriptor{descs[imgReferrer], descs[imgReferrer], descs[index]}

	type args struct {
		ctx          context.Context
		target       oras.ReadOnlyGraphTarget
		desc         ocispec.Descriptor
		artifactType string
	}
	ctx := context.Background()
	tests := []struct {
		name    string
		args    args
		want    []ocispec.Descriptor
		wantErr bool
	}{
		// TODO: Add test cases.
		{"should fail when a referrer lister failed to get referrers", args{ctx, &errLister{}, ocispec.Descriptor{}, ""}, nil, true},
		{"should return referrers when target is a referrer lister", args{ctx, &refLister{referrers: referrers}, ocispec.Descriptor{}, ""}, referrers, false},
		{"should return nil for non-manifest node", args{ctx, &predecessorFinder{}, descs[index], ""}, nil, false},
		{"should return nil for non-manifest node", args{ctx, &predecessorFinder{}, descs[imgConfig], ""}, nil, false},
		{"should return referrers in predecessor", args{ctx, &predecessorFinder{predecessors: predecessors}, descs[subject], ""}, referrers, false},
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
}
