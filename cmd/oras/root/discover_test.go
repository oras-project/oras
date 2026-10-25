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
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras/cmd/oras/internal/display/metadata/tree"
	"oras.land/oras/cmd/oras/internal/option"
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
	calls []string
	err   error
}

func (h *recordingDiscoverHandler) OnDiscovered(_, _ ocispec.Descriptor) error {
	h.count++
	h.calls = append(h.calls, "OnDiscovered")
	return h.err
}

func (h *recordingDiscoverHandler) Render() error {
	h.calls = append(h.calls, "Render")
	return nil
}

func TestNewDiscoverHandler_injected(t *testing.T) {
	want := &recordingDiscoverHandler{}
	got, err := newDiscoverHandler(&discoverOptions{}, ocispec.Descriptor{}, want)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatal("newDiscoverHandler() did not return injected handler")
	}
}

func TestNewDiscoverHandler_default(t *testing.T) {
	opts := &discoverOptions{Format: option.Format{Type: option.FormatTypeTree.Name}}
	got, err := newDiscoverHandler(opts, ocispec.Descriptor{})
	if err != nil {
		t.Fatal(err)
	}
	if reflect.TypeOf(got) != reflect.TypeOf(tree.NewDiscoverHandler(nil, "", ocispec.Descriptor{}, false, nil)) {
		t.Errorf("handler type = %T, want tree handler", got)
	}
}

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

func TestFetchAllReferrers_HandlerErrorStopsTraversal(t *testing.T) {
	wantedErr := errors.New("discover handler failed")
	desc := ocispec.Descriptor{Digest: digest.FromString("root"), MediaType: ocispec.MediaTypeImageManifest}
	target := &cyclicReferrerTarget{referrers: map[digest.Digest][]ocispec.Descriptor{
		desc.Digest: {{Digest: digest.FromString("child"), MediaType: ocispec.MediaTypeImageManifest}},
	}}
	handler := &recordingDiscoverHandler{err: wantedErr}
	if err := fetchAllReferrers(context.Background(), target, desc, "", handler, 0, make(map[digest.Digest]bool)); !errors.Is(err, wantedErr) {
		t.Fatalf("fetchAllReferrers() error = %v, want %v", err, wantedErr)
	}
	if got, want := handler.calls, []string{"OnDiscovered"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("handler calls = %v, want %v", got, want)
	}
}

func TestDiscoverCmd_OutputDeprecationMessage(t *testing.T) {
	for _, format := range []string{"tree", "table", "json"} {
		t.Run(format, func(t *testing.T) {
			cmd := discoverCmd()
			var stderr bytes.Buffer
			cmd.SetErr(&stderr)
			if err := cmd.ParseFlags([]string{"--output", format}); err != nil {
				t.Fatal(err)
			}
			if err := cmd.PreRunE(cmd, []string{"localhost:5000/hello:v1"}); err != nil {
				t.Fatal(err)
			}
			want := "[DEPRECATED] --output is deprecated, try `--format " + format + "` instead"
			if !strings.Contains(stderr.String(), want) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), want)
			}
		})
	}
}

func TestRunDiscover_injectedHandler(t *testing.T) {
	ctx := context.Background()
	layout := t.TempDir()
	store, err := oci.New(layout)
	if err != nil {
		t.Fatal(err)
	}
	desc, err := oras.PackManifest(ctx, store, oras.PackManifestVersion1_1, "application/example", oras.PackManifestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Tag(ctx, desc, "source"); err != nil {
		t.Fatal(err)
	}
	opts := &discoverOptions{Target: option.Target{Type: option.TargetTypeOCILayout, Path: layout, Reference: "source"}}
	handler := &recordingDiscoverHandler{}
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	if err := runDiscover(cmd, opts, handler); err != nil {
		t.Fatal(err)
	}
	if got, want := handler.calls, []string{"Render"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("handler calls = %v, want %v", got, want)
	}
}
