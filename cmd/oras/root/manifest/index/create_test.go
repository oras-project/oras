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

package index

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras/cmd/oras/internal/display/status"
)

type testReadOnlyTarget struct {
	content []byte
}

func (tros *testReadOnlyTarget) Exists(ctx context.Context, desc ocispec.Descriptor) (bool, error) {
	return true, nil
}

func (tros *testReadOnlyTarget) Fetch(ctx context.Context, desc ocispec.Descriptor) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(tros.content)), nil
}

func (tros *testReadOnlyTarget) Resolve(ctx context.Context, reference string) (ocispec.Descriptor, error) {
	if bytes.Equal(tros.content, []byte("index")) {
		return ocispec.Descriptor{MediaType: ocispec.MediaTypeImageIndex, Digest: digest.FromBytes(tros.content), Size: int64(len(tros.content))}, nil
	}
	return ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: digest.FromBytes(tros.content), Size: int64(len(tros.content))}, nil
}

func NewTestReadOnlyTarget(text string) oras.ReadOnlyTarget {
	return &testReadOnlyTarget{content: []byte(text)}
}

type testCreateDisplayStatus struct {
	onFetchingError    bool
	onFetchedError     bool
	onIndexPackedError bool
	onIndexPushedError bool
}

func (tds *testCreateDisplayStatus) OnFetching(manifestRef string) error {
	if tds.onFetchingError {
		return fmt.Errorf("OnFetching error")
	}
	return nil
}

func (tds *testCreateDisplayStatus) OnFetched(manifestRef string, desc ocispec.Descriptor) error {
	if tds.onFetchedError {
		return fmt.Errorf("OnFetched error")
	}
	return nil
}

func (tds *testCreateDisplayStatus) OnIndexPacked(desc ocispec.Descriptor) error {
	if tds.onIndexPackedError {
		return fmt.Errorf("error")
	}
	return nil
}

func (tds *testCreateDisplayStatus) OnIndexPushed(path string) error {
	if tds.onIndexPushedError {
		return fmt.Errorf("error")
	}
	return nil
}

func Test_fetchSourceManifests(t *testing.T) {
	testContext := context.Background()
	tests := []struct {
		name          string
		ctx           context.Context
		displayStatus status.ManifestIndexCreateHandler
		target        oras.ReadOnlyTarget
		sources       []string
		want          []ocispec.Descriptor
		wantErr       bool
	}{
		{
			name:          "OnFetching error",
			ctx:           testContext,
			displayStatus: &testCreateDisplayStatus{onFetchingError: true},
			target:        NewTestReadOnlyTarget("test content"),
			sources:       []string{"test"},
			want:          nil,
			wantErr:       true,
		},
		{
			name:          "OnFetched error",
			ctx:           testContext,
			displayStatus: &testCreateDisplayStatus{onFetchedError: true},
			target:        NewTestReadOnlyTarget("test content"),
			sources:       []string{"test"},
			want:          nil,
			wantErr:       true,
		},
		{
			name:          "getPlatform error",
			ctx:           testContext,
			displayStatus: &testCreateDisplayStatus{},
			target:        NewTestReadOnlyTarget("test content"),
			sources:       []string{"test"},
			want:          nil,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fetchSourceManifests(tt.ctx, tt.displayStatus, tt.target, tt.sources)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchSourceManifests() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchSourceManifests() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_enrichDescriptor(t *testing.T) {
	tests := []struct {
		name              string
		target            oras.ReadOnlyTarget
		manifestBytes     []byte
		manifestMediaType string
		checkDesc         func(t *testing.T, gotDesc, inputDesc ocispec.Descriptor)
		wantErr           bool
	}{
		{
			name:   "child index, valid",
			target: NewTestReadOnlyTarget("(unused)"),
			manifestBytes: []byte(`
				{
					"schemaVersion": 2,
					"mediaType": "application/vnd.oci.image.index.v1+json",
					"artifactType": "application/vnd.example",
					"manifests": [],
					"annotations": {
						"test-only": ""
					}
				}
			`),
			manifestMediaType: "application/vnd.oci.image.index.v1+json",
			checkDesc: func(t *testing.T, gotDesc, inputDesc ocispec.Descriptor) {
				if got, want := gotDesc.ArtifactType, "application/vnd.example"; got != want {
					t.Errorf("ArtifactType = %s, want %s", got, want)
				}
			},
			wantErr: false,
		},
		{
			name:   "child manifest, valid with platform",
			target: NewTestReadOnlyTarget(`{"architecture":"testarch","os":"testos"}`),
			manifestBytes: []byte(`
				{
					"schemaVersion": 2,
					"mediaType": "application/vnd.oci.image.manifest.v1+json",
					"artifactType": "application/vnd.example",
					"config": {
						"mediaType": "application/vnd.oci.image.config.v1+json",
						"digest": "sha256:5fefde2b739e2ff1976ecea4fb4f5e4827a1c424e9b1fb147ba5fd21b9197422",
						"size": 41
					},
					"layers": [],
					"annotations": {
						"test-only": ""
					}
				}
			`),
			manifestMediaType: "application/vnd.oci.image.manifest.v1+json",
			checkDesc: func(t *testing.T, gotDesc, inputDesc ocispec.Descriptor) {
				if got, want := gotDesc.ArtifactType, "application/vnd.example"; got != want {
					t.Errorf("ArtifactType = %s, want %s", got, want)
				}
				wantPlatform := &ocispec.Platform{
					Architecture: "testarch",
					OS:           "testos",
				}
				if !reflect.DeepEqual(gotDesc.Platform, wantPlatform) {
					t.Errorf("Platform = %#v, want %#v", gotDesc.Platform, wantPlatform)
				}
			},
			wantErr: false,
		},
		{
			name:   "child manifest, valid without platform",
			target: NewTestReadOnlyTarget(`intentionally not valid JSON`),
			manifestBytes: []byte(`
				{
					"schemaVersion": 2,
					"mediaType": "application/vnd.oci.image.manifest.v1+json",
					"artifactType": "application/vnd.example",
					"config": {
						"mediaType": "application/vnd.other",
						"digest": "sha256:dc889043956f34871cc04ae96e03efc29dfe2f582c26195a72dd4827f4dd830d",
						"size": 28
					},
					"layers": [],
					"annotations": {
						"test-only": ""
					}
				}
			`),
			manifestMediaType: "application/vnd.oci.image.manifest.v1+json",
			checkDesc: func(t *testing.T, gotDesc, inputDesc ocispec.Descriptor) {
				if got, want := gotDesc.ArtifactType, "application/vnd.example"; got != want {
					t.Errorf("ArtifactType = %s, want %s", got, want)
				}
				if gotDesc.Platform != nil {
					t.Errorf("Platform = %#v, want nil", gotDesc.Platform)
				}
			},
			wantErr: false,
		},
		{
			name:              "child of unrecognized type",
			target:            NewTestReadOnlyTarget("(unused)"),
			manifestBytes:     []byte(`{}`),
			manifestMediaType: "application/vnd.custom",
			checkDesc: func(t *testing.T, gotDesc, inputDesc ocispec.Descriptor) {
				if !reflect.DeepEqual(gotDesc, inputDesc) {
					t.Errorf("result does not match input: got %#v, want %#v", gotDesc, inputDesc)
				}
			},
			wantErr: false,
		},
		{
			name:              "child manifest, invalid",
			target:            NewTestReadOnlyTarget(`unused`),
			manifestBytes:     []byte(`not actually a manifest`),
			manifestMediaType: "application/vnd.oci.image.manifest.v1+json",
			wantErr:           true,
		},
		{
			name:              "child index, invalid",
			target:            NewTestReadOnlyTarget(`unused`),
			manifestBytes:     []byte(`not actually an index`),
			manifestMediaType: "application/vnd.oci.image.index.v1+json",
			wantErr:           true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputDesc := ocispec.Descriptor{
				MediaType: tt.manifestMediaType,
				Digest:    digest.FromBytes(tt.manifestBytes),
				Size:      int64(len(tt.manifestBytes)),
			}
			gotDesc, err := enrichDescriptor(t.Context(), tt.target, inputDesc, tt.manifestBytes)
			if (err != nil) != tt.wantErr {
				t.Fatalf("enrichDescriptor() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return // gotDesc is not valid when there's an error
			}
			// The fields we set in inputDesc should always be unchanged in
			// the result.
			if got, want := gotDesc.MediaType, inputDesc.MediaType; got != want {
				t.Errorf("MediaType = %s, want %s", got, want)
			}
			if got, want := gotDesc.Digest, inputDesc.Digest; got != want {
				t.Errorf("Digest = %s, want %s", got, want)
			}
			if got, want := gotDesc.Size, inputDesc.Size; got != want {
				t.Errorf("Size = %d, want %d", got, want)
			}
			// Currently we do not enrich with annotations, though a future
			// proposal might change that in which case the following should
			// be removed in favor of specific tests in checkDesc.
			//
			// Discussion here:
			//     https://github.com/oras-project/oras/pull/1696#issuecomment-2852473626
			if len(gotDesc.Annotations) != 0 {
				t.Errorf("Annotations = %#v, want none", gotDesc.Annotations)
			}
			// Other test-case-specific checks
			if tt.checkDesc != nil {
				tt.checkDesc(t, gotDesc, inputDesc)
			}
		})
	}
}

func Test_validateMediaType(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"application/json", true},
		{ocispec.MediaTypeEmptyJSON, true},
		{ocispec.MediaTypeImageManifest, true},
		{ocispec.MediaTypeImageIndex, true},
		{"application/vnd.custom", true},
		{"", false},
		{"json", false},
		{"application/-json", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := validateMediaType(tt.input)
			if err == nil && !tt.valid {
				t.Errorf("no error for invalid media type %q", tt.input)
			}
			if err != nil && tt.valid {
				t.Errorf("unexpected error for valid media type %q: %s", tt.input, err)
			}
		})
	}
}
