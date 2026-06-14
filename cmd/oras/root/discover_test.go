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

package root

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// cyclicReferrerTarget is a minimal ReadOnlyGraphTarget that serves a
// configurable referrer graph via the Referrers API. It lets tests model a
// malicious registry that returns a cyclic referrer graph.
type cyclicReferrerTarget struct {
	referrers map[digest.Digest][]ocispec.Descriptor
}

func (t *cyclicReferrerTarget) Referrers(_ context.Context, desc ocispec.Descriptor, _ string, fn func(referrers []ocispec.Descriptor) error) error {
	return fn(t.referrers[desc.Digest])
}

func (t *cyclicReferrerTarget) Fetch(context.Context, ocispec.Descriptor) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (t *cyclicReferrerTarget) Exists(context.Context, ocispec.Descriptor) (bool, error) {
	return false, nil
}

func (t *cyclicReferrerTarget) Predecessors(context.Context, ocispec.Descriptor) ([]ocispec.Descriptor, error) {
	return nil, nil
}

func (t *cyclicReferrerTarget) Resolve(context.Context, string) (ocispec.Descriptor, error) {
	return ocispec.Descriptor{}, errors.New("not implemented")
}

// recordingDiscoverHandler counts OnDiscovered calls.
type recordingDiscoverHandler struct {
	count int
}

func (h *recordingDiscoverHandler) OnDiscovered(_, _ ocispec.Descriptor) error {
	h.count++
	return nil
}

func (h *recordingDiscoverHandler) Render() error { return nil }

func TestFetchAllReferrers_CyclicGraphTerminates(t *testing.T) {
	descA := ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: digest.FromString("A"), Size: 1}
	descB := ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: digest.FromString("B"), Size: 1}
	// A -> B -> A, a cycle a malicious registry could craft.
	target := &cyclicReferrerTarget{referrers: map[digest.Digest][]ocispec.Descriptor{
		descA.Digest: {descB},
		descB.Digest: {descA},
	}}
	handler := &recordingDiscoverHandler{}

	// depth 0 means unlimited; without cycle detection this never returns.
	if err := fetchAllReferrers(context.Background(), target, descA, "", handler, 0, make(map[digest.Digest]bool)); err != nil {
		t.Fatalf("fetchAllReferrers() error = %v", err)
	}
	// Each edge (A->B and B->A) is reported exactly once.
	if handler.count != 2 {
		t.Errorf("OnDiscovered called %d times, want 2", handler.count)
	}
}
