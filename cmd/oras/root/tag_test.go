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

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras/cmd/oras/internal/display/displaytest"
	"oras.land/oras/cmd/oras/internal/display/metadata/text"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/listener"
)

func Test_tagCmd_requiresSourceAndDestination(t *testing.T) {
	rootCmd := &cobra.Command{Use: "oras"}
	rootCmd.AddCommand(tagCmd())
	rootCmd.SetArgs([]string{"tag", "localhost:5000/hello:v1"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an argument validation error")
	}
	if !strings.Contains(err.Error(), "at least 2 arguments") {
		t.Fatalf("expected an argument count error, got %v", err)
	}
}

func TestNewTagHandler_injected(t *testing.T) {
	want := &displaytest.RecordingTagHandler{Recorder: &displaytest.CallRecorder{}}
	if got := newTagHandler(&tagOptions{}, want); got != want {
		t.Fatal("newTagHandler() did not return injected handler")
	}
}

func TestNewTagHandler_default(t *testing.T) {
	opts := &tagOptions{Target: option.Target{Reference: "example:v1"}}
	if got, want := reflect.TypeOf(newTagHandler(opts)), reflect.TypeOf(text.NewTagHandler(nil, opts.Target)); got != want {
		t.Errorf("handler type = %v, want %v", got, want)
	}
}

func TestTagListener_handlerCallOrderAndError(t *testing.T) {
	target := memory.New()
	desc := ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest}
	wantedErr := errors.New("tagging failed")
	recorder := &displaytest.CallRecorder{}
	handler := &displaytest.RecordingTagHandler{Recorder: recorder, TaggingErr: wantedErr}
	wrapped := listener.NewTagListener(target, handler.OnTagging, handler.OnTagged)
	if err := wrapped.Tag(context.Background(), desc, "v1"); !errors.Is(err, wantedErr) {
		t.Fatalf("Tag() error = %v, want %v", err, wantedErr)
	}
	if got, want := recorder.Snapshot(), []string{"OnTagging"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("handler calls = %v, want %v", got, want)
	}
}

func TestTagWithHandler_handlerCallOrderAndError(t *testing.T) {
	ctx := context.Background()
	target := memory.New()
	desc, err := oras.PackManifest(ctx, target, oras.PackManifestVersion1_1, "application/example", oras.PackManifestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := target.Tag(ctx, desc, "source"); err != nil {
		t.Fatal(err)
	}
	wantedErr := errors.New("tagging failed")
	recorder := &displaytest.CallRecorder{}
	handler := &displaytest.RecordingTagHandler{Recorder: recorder, TaggingErr: wantedErr}
	opts := oras.DefaultTagNOptions
	err = tagWithHandler(ctx, target, "source", []string{"destination"}, opts, handler)
	if !errors.Is(err, wantedErr) {
		t.Fatalf("tagWithHandler() error = %v, want %v", err, wantedErr)
	}
	if got, want := recorder.Snapshot(), []string{"OnTagging"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("handler calls = %v, want %v", got, want)
	}
}

func TestTagWithHandler_handlerCallOrder(t *testing.T) {
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
	handler := &displaytest.RecordingTagHandler{Recorder: recorder}
	err = tagWithHandler(ctx, target, "source", []string{"destination"}, oras.DefaultTagNOptions, handler)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := recorder.Snapshot(), []string{"OnTagging", "OnTagged"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("handler calls = %v, want %v", got, want)
	}
}

func TestTagManifest_injectedHandler(t *testing.T) {
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
	opts := &tagOptions{Target: option.Target{Type: option.TargetTypeOCILayout, Path: layout, Reference: "source", RawReference: "source"}, targetRefs: []string{"destination"}}
	recorder := &displaytest.CallRecorder{}
	handler := &displaytest.RecordingTagHandler{Recorder: recorder}
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	if err := tagManifest(cmd, opts, handler); err != nil {
		t.Fatal(err)
	}
	if got, want := recorder.Snapshot(), []string{"OnTagging", "OnTagged"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("handler calls = %v, want %v", got, want)
	}
}
