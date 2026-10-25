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

package manifest

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras/cmd/oras/internal/display/metadata/text"
	"oras.land/oras/cmd/oras/internal/display/status"
	"oras.land/oras/cmd/oras/internal/option"
)

type recordingManifestPushStatusHandler struct {
	calls []string
	err   error
}

func (h *recordingManifestPushStatusHandler) OnManifestPushSkipped() error {
	h.calls = append(h.calls, "OnManifestPushSkipped")
	return h.err
}
func (h *recordingManifestPushStatusHandler) OnManifestPushing() error {
	h.calls = append(h.calls, "OnManifestPushing")
	return h.err
}
func (h *recordingManifestPushStatusHandler) OnManifestPushed() error {
	h.calls = append(h.calls, "OnManifestPushed")
	return h.err
}

type recordingManifestPushMetadataHandler struct {
	calls []string
	err   error
}

type failingManifestPushTarget struct {
	*memory.Store
	err error
}

func (t *failingManifestPushTarget) Push(context.Context, ocispec.Descriptor, io.Reader) error {
	return t.err
}

func (h *recordingManifestPushMetadataHandler) OnTagged(ocispec.Descriptor, string) error {
	h.calls = append(h.calls, "OnTagged")
	return h.err
}
func (h *recordingManifestPushMetadataHandler) OnManifestPushed(ocispec.Descriptor) error {
	h.calls = append(h.calls, "OnManifestPushed")
	return h.err
}
func (h *recordingManifestPushMetadataHandler) Render() error {
	h.calls = append(h.calls, "Render")
	return h.err
}

func Test_pushManifestCmd_invalidExtraTag(t *testing.T) {
	cmd := pushCmd()
	cmd.SetArgs([]string{
		"localhost:5000/hello:v1,,invalid@@tag",
		"manifest.json",
	})

	err := cmd.Execute()
	if err == nil || err.Error() != "empty extra tag" {
		t.Fatalf("got %v, want empty extra tag error", err)
	}
}

func Test_pushManifestCmd_invalidHeader(t *testing.T) {
	cmd := pushCmd()
	cmd.SetArgs([]string{
		"localhost:5000/hello:v1",
		"manifest.json",
		"--header", "invalid",
	})

	err := cmd.Execute()
	if err == nil || err.Error() != `invalid header: "invalid"` {
		t.Fatalf("got %v, want invalid header error", err)
	}
}

func TestNewManifestPushHandlers_injected(t *testing.T) {
	statusHandler := &recordingManifestPushStatusHandler{}
	metadataHandler := &recordingManifestPushMetadataHandler{}
	gotStatus, gotMetadata := newManifestPushHandlers(pushOptions{}, ocispec.Descriptor{}, manifestPushHandlers{statusHandler, metadataHandler})
	if gotStatus != statusHandler || gotMetadata != metadataHandler {
		t.Fatal("newManifestPushHandlers() did not return injected handlers")
	}
}

func TestNewManifestPushHandlers_default(t *testing.T) {
	opts := pushOptions{Target: option.Target{Reference: "example:v1"}}
	gotStatus, gotMetadata := newManifestPushHandlers(opts, ocispec.Descriptor{})
	if reflect.TypeOf(gotStatus) != reflect.TypeOf(status.NewTextManifestPushHandler(nil, ocispec.Descriptor{})) {
		t.Errorf("status handler type = %T, want text manifest push handler", gotStatus)
	}
	if reflect.TypeOf(gotMetadata) != reflect.TypeOf(text.NewManifestPushHandler(nil, &opts.Target)) {
		t.Errorf("metadata handler type = %T, want text manifest push handler", gotMetadata)
	}
}

func TestRecordingManifestPushHandlers_errorStopsSequence(t *testing.T) {
	wantedErr := errors.New("manifest push failed")
	statusHandler := &recordingManifestPushStatusHandler{err: wantedErr}
	if err := statusHandler.OnManifestPushing(); !errors.Is(err, wantedErr) {
		t.Fatalf("OnManifestPushing() error = %v, want %v", err, wantedErr)
	}
	if got, want := statusHandler.calls, []string{"OnManifestPushing"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("status calls = %v, want %v", got, want)
	}
}

func TestPushManifestToTarget_handlerCallOrderAndError(t *testing.T) {
	ctx := context.Background()
	target := memory.New()
	wantedErr := errors.New("manifest push failed")
	statusHandler := &recordingManifestPushStatusHandler{err: wantedErr}
	metadataHandler := &recordingManifestPushMetadataHandler{}
	err := pushManifestToTarget(ctx, target, "application/vnd.oci.image.manifest.v1+json", []byte(`{"schemaVersion":2}`), "v1", false, statusHandler)
	if !errors.Is(err, wantedErr) {
		t.Fatalf("pushManifestToTarget() error = %v, want %v", err, wantedErr)
	}
	if got, want := statusHandler.calls, []string{"OnManifestPushing"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("status calls = %v, want %v", got, want)
	}
	if len(metadataHandler.calls) != 0 {
		t.Fatalf("metadata calls = %v, want none", metadataHandler.calls)
	}

	statusHandler.err = nil
	if err := pushManifestToTarget(ctx, target, "application/vnd.oci.image.manifest.v1+json", []byte(`{"schemaVersion":2}`), "v1", true, statusHandler); err != nil {
		t.Fatal(err)
	}
	if got, want := statusHandler.calls, []string{"OnManifestPushing", "OnManifestPushSkipped"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("status calls = %v, want %v", got, want)
	}
	statusHandler.calls = nil
	if err := pushManifestToTarget(ctx, target, "application/vnd.oci.image.manifest.v1+json", []byte(`{"schemaVersion":2}`), "v2", false, statusHandler); err != nil {
		t.Fatal(err)
	}
	if got, want := statusHandler.calls, []string{"OnManifestPushing", "OnManifestPushed"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("status calls = %v, want %v", got, want)
	}
}

func TestPushManifestToTarget_targetError(t *testing.T) {
	wantedErr := errors.New("target push failed")
	statusHandler := &recordingManifestPushStatusHandler{}
	target := &failingManifestPushTarget{Store: memory.New(), err: wantedErr}
	err := pushManifestToTarget(context.Background(), target, "application/vnd.oci.image.manifest.v1+json", []byte(`{"schemaVersion":2}`), "v1", false, statusHandler)
	if !errors.Is(err, wantedErr) {
		t.Fatalf("pushManifestToTarget() error = %v, want %v", err, wantedErr)
	}
	if got, want := statusHandler.calls, []string{"OnManifestPushing"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("status calls = %v, want %v", got, want)
	}
}

func TestPushManifest_injectedHandlers(t *testing.T) {
	layout := t.TempDir()
	fileRef := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(fileRef, []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"}`), 0600); err != nil {
		t.Fatal(err)
	}
	opts := pushOptions{Target: option.Target{Type: option.TargetTypeOCILayout, Path: layout, Reference: "source"}, fileRef: fileRef}
	statusHandler := &recordingManifestPushStatusHandler{}
	metadataHandler := &recordingManifestPushMetadataHandler{}
	cmd := &cobra.Command{Use: "manifest push"}
	cmd.SetContext(context.Background())
	if err := pushManifest(cmd, opts, manifestPushHandlers{statusHandler, metadataHandler}); err != nil {
		t.Fatal(err)
	}
	if got, want := statusHandler.calls, []string{"OnManifestPushing", "OnManifestPushed"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("status calls = %v, want %v", got, want)
	}
	if got, want := metadataHandler.calls, []string{"OnManifestPushed", "Render"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("metadata calls = %v, want %v", got, want)
	}
}

func TestPushManifest_injectedStatusError(t *testing.T) {
	layout := t.TempDir()
	fileRef := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(fileRef, []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"}`), 0600); err != nil {
		t.Fatal(err)
	}
	opts := pushOptions{Target: option.Target{Type: option.TargetTypeOCILayout, Path: layout, Reference: "source"}, fileRef: fileRef}
	wantedErr := errors.New("status failed")
	statusHandler := &recordingManifestPushStatusHandler{err: wantedErr}
	metadataHandler := &recordingManifestPushMetadataHandler{}
	cmd := &cobra.Command{Use: "manifest push"}
	cmd.SetContext(context.Background())
	if err := pushManifest(cmd, opts, manifestPushHandlers{statusHandler, metadataHandler}); !errors.Is(err, wantedErr) {
		t.Fatalf("pushManifest() error = %v, want %v", err, wantedErr)
	}
	if got, want := statusHandler.calls, []string{"OnManifestPushing"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("status calls = %v, want %v", got, want)
	}
}
