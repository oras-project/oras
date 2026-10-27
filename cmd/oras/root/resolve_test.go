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
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras/cmd/oras/internal/display/displaytest"
	"oras.land/oras/cmd/oras/internal/display/metadata/text"
	"oras.land/oras/cmd/oras/internal/option"
)

func TestNewResolveHandler_injected(t *testing.T) {
	want := &displaytest.RecordingResolveHandler{Recorder: &displaytest.CallRecorder{}}
	got := newResolveHandler(&resolveOptions{}, want)
	if got != want {
		t.Fatal("newResolveHandler() did not return injected handler")
	}
}

func TestNewResolveHandler_default(t *testing.T) {
	opts := &resolveOptions{Common: option.Common{}, fullRef: true}
	got := newResolveHandler(opts)
	if reflect.TypeOf(got) != reflect.TypeOf(text.NewResolveHandler(nil, true, "")) {
		t.Errorf("handler type = %T, want text resolve handler", got)
	}
}

func TestResolveWithHandler_callsInjectedHandler(t *testing.T) {
	ctx := context.Background()
	target := memory.New()
	desc, err := oras.PackManifest(ctx, target, oras.PackManifestVersion1_1, "application/example", oras.PackManifestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := target.Tag(ctx, desc, "source"); err != nil {
		t.Fatal(err)
	}
	recorder := &displaytest.CallRecorder{}
	handler := &displaytest.RecordingResolveHandler{Recorder: recorder, Err: errors.New("resolve output failed")}
	err = resolveWithHandler(ctx, target, "source", oras.DefaultResolveOptions, handler)
	if !errors.Is(err, handler.Err) {
		t.Fatalf("resolveWithHandler() error = %v, want %v", err, handler.Err)
	}
	if got, want := recorder.Snapshot(), []string{"OnResolved"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("handler calls = %v, want %v", got, want)
	}
	if desc.Digest == "" {
		t.Fatal("PackManifest returned empty digest")
	}
}

func TestResolveWithHandler_resolveError(t *testing.T) {
	err := resolveWithHandler(context.Background(), memory.New(), "missing", oras.DefaultResolveOptions, &displaytest.RecordingResolveHandler{Recorder: &displaytest.CallRecorder{}})
	if err == nil || !strings.Contains(err.Error(), "failed to resolve digest") {
		t.Fatalf("resolveWithHandler() error = %v, want wrapped resolve error", err)
	}
}

func TestRunResolve_injectedHandler(t *testing.T) {
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
	opts := &resolveOptions{Target: option.Target{Type: option.TargetTypeOCILayout, Path: layout, Reference: "source"}}
	recorder := &displaytest.CallRecorder{}
	handler := &displaytest.RecordingResolveHandler{Recorder: recorder}
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	if err := runResolve(cmd, opts, handler); err != nil {
		t.Fatal(err)
	}
	if got, want := recorder.Snapshot(), []string{"OnResolved"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("handler calls = %v, want %v", got, want)
	}
}
