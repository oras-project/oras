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
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras/cmd/oras/internal/display/status"
)

type testUpdateDisplayStatus struct {
	onFetchingError        bool
	onFetchedError         bool
	onIndexPackedError     bool
	onIndexPushedError     bool
	onManifestRemovedError bool
	onManifestAddedError   bool
	onIndexMergedError     bool
}

func (tds *testUpdateDisplayStatus) OnFetching(manifestRef string) error {
	if tds.onFetchingError {
		return fmt.Errorf("OnFetching error")
	}
	return nil
}

func (tds *testUpdateDisplayStatus) OnFetched(manifestRef string, desc ocispec.Descriptor) error {
	if tds.onFetchedError {
		return fmt.Errorf("OnFetched error")
	}
	return nil
}

func (tds *testUpdateDisplayStatus) OnIndexPacked(desc ocispec.Descriptor) error {
	if tds.onIndexPackedError {
		return fmt.Errorf("error")
	}
	return nil
}

func (tds *testUpdateDisplayStatus) OnIndexPushed(path string) error {
	if tds.onIndexPushedError {
		return fmt.Errorf("error")
	}
	return nil
}

func (tds *testUpdateDisplayStatus) OnManifestRemoved(digest digest.Digest) error {
	if tds.onManifestRemovedError {
		return fmt.Errorf("OnManifestRemoved error")
	}
	return nil
}

func (tds *testUpdateDisplayStatus) OnManifestAdded(manifestRef string, desc ocispec.Descriptor) error {
	if tds.onManifestAddedError {
		return fmt.Errorf("error")
	}
	return nil
}

func (tds *testUpdateDisplayStatus) OnIndexMerged(indexRef string, desc ocispec.Descriptor) error {
	if tds.onIndexMergedError {
		return fmt.Errorf("error")
	}
	return nil
}

var (
	A = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Size:      16,
		Digest:    "sha256:58efe73e78fe043ca31b89007a025c594ce12aa7e6da27d21c7b14b50112e255",
	}
	B = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Size:      18,
		Digest:    "sha256:9d16f5505246424aed7116cb21216704ba8c919997d0f1f37e154c11d509e1d2",
	}
	C = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Size:      19,
		Digest:    "sha256:fd6ed2f36b5465244d5dc86cb4e7df0ab8a9d24adc57825099f522fe009a22bb",
	}
)

func Test_doRemoveManifests(t *testing.T) {
	tests := []struct {
		name          string
		manifests     []ocispec.Descriptor
		digestSet     map[digest.Digest]bool
		displayStatus status.ManifestIndexUpdateHandler
		indexRef      string
		want          []ocispec.Descriptor
		wantErr       bool
	}{
		{
			name:          "remove one matched item",
			manifests:     []ocispec.Descriptor{A, B, C},
			digestSet:     map[digest.Digest]bool{B.Digest: false},
			displayStatus: &testUpdateDisplayStatus{},
			indexRef:      "test01",
			want:          []ocispec.Descriptor{A, C},
			wantErr:       false,
		},
		{
			name:          "remove all matched items",
			manifests:     []ocispec.Descriptor{A, B, A, C, A, A, A},
			digestSet:     map[digest.Digest]bool{A.Digest: false},
			displayStatus: &testUpdateDisplayStatus{},
			indexRef:      "test02",
			want:          []ocispec.Descriptor{B, C},
			wantErr:       false,
		},
		{
			name:          "remove correctly when there is only one item",
			manifests:     []ocispec.Descriptor{A},
			digestSet:     map[digest.Digest]bool{A.Digest: false},
			displayStatus: &testUpdateDisplayStatus{},
			indexRef:      "test03",
			want:          []ocispec.Descriptor{},
			wantErr:       false,
		},
		{
			name:          "remove multiple distinct manifests",
			manifests:     []ocispec.Descriptor{A, B, C},
			digestSet:     map[digest.Digest]bool{A.Digest: false, C.Digest: false},
			displayStatus: &testUpdateDisplayStatus{},
			indexRef:      "test04",
			want:          []ocispec.Descriptor{B},
			wantErr:       false,
		},
		{
			name:          "remove multiple duplicate manifests",
			manifests:     []ocispec.Descriptor{A, B, C, C, B, A, B},
			digestSet:     map[digest.Digest]bool{A.Digest: false, C.Digest: false},
			displayStatus: &testUpdateDisplayStatus{},
			indexRef:      "test05",
			want:          []ocispec.Descriptor{B, B, B},
			wantErr:       false,
		},
		{
			name:          "return error when deleting a nonexistent item",
			manifests:     []ocispec.Descriptor{A, C},
			digestSet:     map[digest.Digest]bool{B.Digest: false},
			displayStatus: &testUpdateDisplayStatus{},
			indexRef:      "test06",
			want:          nil,
			wantErr:       true,
		},
		{
			name:          "handler error",
			manifests:     []ocispec.Descriptor{A, B, C},
			digestSet:     map[digest.Digest]bool{B.Digest: false},
			displayStatus: &testUpdateDisplayStatus{onManifestRemovedError: true},
			indexRef:      "test07",
			want:          nil,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := doRemoveManifests(tt.manifests, tt.digestSet, tt.displayStatus, tt.indexRef)
			if (err != nil) != tt.wantErr {
				t.Errorf("removeManifestsFromIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("removeManifestsFromIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fetchIndex(t *testing.T) {
	testContext := context.Background()
	tests := []struct {
		name      string
		ctx       context.Context
		handler   status.ManifestIndexUpdateHandler
		target    oras.ReadOnlyTarget
		reference string
		want      ocispec.Index
		wantErr   bool
	}{
		{
			name:      "OnFetching error",
			ctx:       testContext,
			handler:   &testUpdateDisplayStatus{onFetchingError: true},
			target:    NewTestReadOnlyTarget("index"),
			reference: "test",
			want:      ocispec.Index{},
			wantErr:   true,
		},
		{
			name:      "OnFetched error",
			ctx:       testContext,
			handler:   &testUpdateDisplayStatus{onFetchedError: true},
			target:    NewTestReadOnlyTarget("index"),
			reference: "test",
			want:      ocispec.Index{},
			wantErr:   true,
		},
		{
			name:      "Unmarshall error",
			ctx:       testContext,
			handler:   &testUpdateDisplayStatus{},
			target:    NewTestReadOnlyTarget("index"),
			reference: "test",
			want:      ocispec.Index{},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fetchIndex(tt.ctx, tt.handler, tt.target, tt.reference)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mergeIndexes(t *testing.T) {
	testContext := context.Background()
	tests := []struct {
		name           string
		ctx            context.Context
		displayStatus  status.ManifestIndexUpdateHandler
		manifests      []ocispec.Descriptor
		target         oras.ReadOnlyTarget
		mergeArguments []string
		want           []ocispec.Descriptor
		wantErr        bool
	}{
		{
			name:           "OnFetching error",
			ctx:            testContext,
			displayStatus:  &testUpdateDisplayStatus{onFetchingError: true},
			manifests:      []ocispec.Descriptor{},
			target:         NewTestReadOnlyTarget("index"),
			mergeArguments: []string{"test"},
			want:           nil,
			wantErr:        true,
		},
		{
			name:           "OnFetched error",
			ctx:            testContext,
			displayStatus:  &testUpdateDisplayStatus{onFetchedError: true},
			manifests:      []ocispec.Descriptor{},
			target:         NewTestReadOnlyTarget("index"),
			mergeArguments: []string{"test"},
			want:           nil,
			wantErr:        true,
		},
		{
			name:           "unmarshall error",
			ctx:            testContext,
			displayStatus:  &testUpdateDisplayStatus{},
			manifests:      []ocispec.Descriptor{},
			target:         NewTestReadOnlyTarget("index"),
			mergeArguments: []string{"test"},
			want:           nil,
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeIndexes(tt.ctx, tt.displayStatus, tt.manifests, tt.target, tt.mergeArguments)
			if (err != nil) != tt.wantErr {
				t.Errorf("mergeIndexes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeIndexes() = %v, want %v", got, tt.want)
			}
		})
	}
}
