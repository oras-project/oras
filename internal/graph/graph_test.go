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
	"reflect"
	"strconv"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras/internal/docker"
	"oras.land/oras/internal/testutils"
)

func TestSuccessors(t *testing.T) {
	mockFetcher := testutils.NewMockFetcher()
	fetcher := mockFetcher.Fetcher
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
		{"should get successors of a docker image", args{ctx, fetcher, mockFetcher.DockerImage}, nil, nil, &mockFetcher.Config, false},
		{"should get successors of an OCI image", args{ctx, fetcher, mockFetcher.OciImage}, []ocispec.Descriptor{mockFetcher.ImageLayer}, &mockFetcher.Subject, &mockFetcher.Config, false},
		{"should get successors of an index", args{ctx, fetcher, mockFetcher.Index}, []ocispec.Descriptor{mockFetcher.Subject}, nil, nil, false},
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

func TestDescriptor_GetSuccessors(t *testing.T) {
	mockFetcher := testutils.NewMockFetcher()

	allFilter := func(ocispec.Descriptor) bool {
		return true
	}
	got, err := FilteredSuccessors(context.Background(), mockFetcher.OciImage, mockFetcher.Fetcher, allFilter)
	if nil != err {
		t.Errorf("FilteredSuccessors unexpected error %v", err)
	}
	if len(got) != 3 {
		t.Errorf("Expected 2 successors got %v", len(got))
	}
	if mockFetcher.Subject.Digest != got[0].Digest {
		t.Errorf("FilteredSuccessors got %v, want %v", got[0], mockFetcher.Subject)
	}
	if mockFetcher.Config.Digest != got[1].Digest {
		t.Errorf("FilteredSuccessors got %v, want %v", got[1], mockFetcher.Subject)
	}

	noConfig := func(desc ocispec.Descriptor) bool {
		return desc.Digest != mockFetcher.Config.Digest
	}
	got, err = FilteredSuccessors(context.Background(), mockFetcher.OciImage, mockFetcher.Fetcher, noConfig)
	if nil != err {
		t.Errorf("FilteredSuccessors unexpected error %v", err)
	}
	if len(got) != 2 {
		t.Errorf("Expected 1 successors got %v", len(got))
	}
	if mockFetcher.Subject.Digest != got[0].Digest {
		t.Errorf("FilteredSuccessors got %v, want %v", got[0], mockFetcher.Subject)
	}

	got, err = FilteredSuccessors(context.Background(), ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest}, mockFetcher.Fetcher, allFilter)
	if nil == err {
		t.Error("FilteredSuccessors expected error")
	}
	if got != nil {
		t.Errorf("FilteredSuccessors unexpected %v", got)
	}
}

func TestFindPredecessors(t *testing.T) {
	store := memory.New()
	ctx := context.Background()

	// prepare subjects
	var subjects []ocispec.Descriptor
	for i := range 2 {
		subject, err := oras.PackManifest(ctx, store, oras.PackManifestVersion1_1, "test/subject", oras.PackManifestOptions{
			ManifestAnnotations: map[string]string{
				"test.subject.number": strconv.Itoa(i),
			},
		})
		if err != nil {
			t.Fatalf("PackManifest unexpected error: %v", err)
		}
		subjects = append(subjects, subject)
	}

	// prepare referrers
	referrers := make(map[digest.Digest]ocispec.Descriptor)
	for _, subject := range subjects {
		for i := range 3 {
			referrer, err := oras.PackManifest(ctx, store, oras.PackManifestVersion1_1, "test/referrer", oras.PackManifestOptions{
				Subject: &subject,
				ManifestAnnotations: map[string]string{
					"test.referrer.number": strconv.Itoa(i),
				},
			})
			if err != nil {
				t.Fatalf("PackManifest unexpected error: %v", err)
			}
			referrers[referrer.Digest] = referrer
		}
	}

	// test FindPredecessors
	opts := oras.DefaultExtendedCopyOptions
	gotReferrers, err := FindPredecessors(ctx, store, subjects, opts)
	if err != nil {
		t.Fatalf("FindPredecessors unexpected error: %v", err)
	}
	if len(gotReferrers) != len(referrers) {
		t.Fatalf("FindPredecessors got %d referrers, want %d", len(gotReferrers), len(referrers))
	}
	for _, got := range gotReferrers {
		wantReferrer, ok := referrers[got.Digest]
		if !ok {
			t.Errorf("FindPredecessors got unexpected referrer %v", got)
		}
		if !reflect.DeepEqual(got, wantReferrer) {
			t.Errorf("FindPredecessors got referrer %v, want %v", got, wantReferrer)
		}
	}
}
