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
