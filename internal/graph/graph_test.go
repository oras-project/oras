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
	"errors"
	"reflect"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
)

var (
	artifactReferrerDesc = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeArtifactManifest,
		Size:      123,
		Digest:    digest.FromBytes([]byte{}),
	}
	imageReferrerDesc = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Size:      123,
		Digest:    digest.FromBytes([]byte{}),
	}
	indexDesc = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageIndex,
		Size:      123,
		Digest:    digest.FromBytes([]byte{}),
	}
	subject = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeArtifactManifest,
		Size:      123,
		Digest:    digest.FromBytes([]byte{}),
	}
	referrers    = []ocispec.Descriptor{artifactReferrerDesc, imageReferrerDesc}
	predecessors = []ocispec.Descriptor{artifactReferrerDesc}
)

type errLister struct {
	oras.ReadOnlyGraphTarget
}

func (e *errLister) Referrers(ctx context.Context, desc ocispec.Descriptor, artifactType string, fn func(referrers []ocispec.Descriptor) error) error {
	return errors.New("")
}

type refLister struct {
	oras.ReadOnlyGraphTarget
}

func (m *refLister) Referrers(ctx context.Context, desc ocispec.Descriptor, artifactType string, fn func(referrers []ocispec.Descriptor) error) error {
	return fn(referrers)
}

type predecessorFinder struct {
	oras.ReadOnlyGraphTarget
}

func (m *predecessorFinder) Predecessors(ctx context.Context, node ocispec.Descriptor) ([]ocispec.Descriptor, error) {
	return predecessors, nil
}

func TestReferrers(t *testing.T) {
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
		{"should return referrers when target is a referrer lister", args{ctx, &refLister{}, ocispec.Descriptor{}, ""}, referrers, false},
		{"should return nil for non-manifest node", args{ctx, &predecessorFinder{}, indexDesc, ""}, nil, false},
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
