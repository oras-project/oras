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
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"

	"oras.land/oras/cmd/oras/internal/display/displaytest"
	"oras.land/oras/cmd/oras/internal/display/metadata/text"
	"oras.land/oras/cmd/oras/internal/display/status"
	"oras.land/oras/cmd/oras/internal/option"
)

func Test_attachCmd_configAndPlatformMutuallyExclusive(t *testing.T) {
	cmd := attachCmd()
	cmd.SetArgs([]string{
		"--artifact-type", "doc/example",
		"--config", "config.json",
		"--platform", "linux/amd64",
		"localhost:5000/hello:v1",
		"hi.txt",
	})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "cannot be used at the same time") {
		t.Fatalf("expected mutual exclusion error, got %v", err)
	}
}

func Test_attachCmd_configFlagRegistered(t *testing.T) {
	cmd := attachCmd()
	f := cmd.Flags().Lookup("config")
	if f == nil {
		t.Fatal("expected --config flag to be registered")
	}
	if f.Usage != "`path` of image config file" {
		t.Fatalf("unexpected usage: %q", f.Usage)
	}
}

func Test_runAttach_errType(t *testing.T) {
	// prepare
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// test
	opts := &attachOptions{
		Packer: option.Packer{
			Annotation: option.Annotation{
				ManifestAnnotations: []string{"one", "two"},
			},
			AnnotationFilePath: "/tmp/whatever",
		},
	}
	got := opts.Packer.Parse(cmd).Error()
	want := errors.New("`--annotation` and `--annotation-file` cannot be both specified").Error()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func Test_attachCmd_configAnnotationsWiredToPackOpts(t *testing.T) {
	// Regression for #2022: config annotations from an annotation file must
	// flow through to oras.PackManifestOptions.ConfigAnnotations.
	annotationFile := filepath.Join(t.TempDir(), "annot.json")
	content := `{"$config":{"org.example.key":"val"}}`
	if err := os.WriteFile(annotationFile, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	opts := &attachOptions{}
	opts.AnnotationFilePath = annotationFile
	if err := opts.Packer.Parse(cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	packOpts := buildAttachPackOpts(opts, ocispec.Descriptor{}, nil)
	if packOpts.ConfigAnnotations == nil {
		t.Fatal("expected ConfigAnnotations to be populated; got nil")
	}
	if packOpts.ConfigAnnotations["org.example.key"] != "val" {
		t.Fatalf("expected 'org.example.key'='val', got %v", packOpts.ConfigAnnotations)
	}
}

func TestNewAttachHandlers_injected(t *testing.T) {
	recorder := &displaytest.CallRecorder{}
	want := attachHandlers{
		status:   &displaytest.RecordingAttachStatusHandler{Recorder: recorder},
		metadata: &displaytest.RecordingAttachMetadataHandler{Recorder: recorder},
	}
	got, err := newAttachHandlers(&attachOptions{}, nil, want)
	if err != nil {
		t.Fatal(err)
	}
	if got.status != want.status || got.metadata != want.metadata {
		t.Fatal("newAttachHandlers() did not return injected handlers")
	}
}

func TestNewAttachHandlers_default(t *testing.T) {
	got, err := newAttachHandlers(&attachOptions{Format: option.Format{Type: option.FormatTypeText.Name}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.TypeOf(got.status) != reflect.TypeOf(status.NewTextAttachHandler(nil, nil)) {
		t.Errorf("status handler type = %T, want text attach handler", got.status)
	}
	if reflect.TypeOf(got.metadata) != reflect.TypeOf(text.NewAttachHandler(nil)) {
		t.Errorf("metadata handler type = %T, want text attach handler", got.metadata)
	}
}

func TestNewAttachHandlers_invalidFormat(t *testing.T) {
	_, err := newAttachHandlers(&attachOptions{Format: option.Format{Type: "invalid"}}, nil)
	if err == nil {
		t.Fatal("newAttachHandlers() error = nil, want invalid format error")
	}
}

func TestRecordingAttachHandlers_callOrderAndError(t *testing.T) {
	recorder := &displaytest.CallRecorder{}
	statusHandler := &displaytest.RecordingAttachStatusHandler{Recorder: recorder, TrackErr: errors.New("track failed")}
	metadataHandler := &displaytest.RecordingAttachMetadataHandler{Recorder: recorder}
	if _, _, err := statusHandler.TrackTarget(nil); err == nil {
		t.Fatal("TrackTarget() error = nil, want error")
	}
	metadataHandler.OnAttached(nil, ocispec.Descriptor{}, ocispec.Descriptor{})
	if got, want := recorder.Snapshot(), []string{"TrackTarget", "OnAttached"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("handler calls = %v, want %v", got, want)
	}
}

func TestRunAttach_injectedHandlers(t *testing.T) {
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
	fileRef := filepath.Join(t.TempDir(), "layer.txt")
	if err := os.WriteFile(fileRef, []byte("layer"), 0600); err != nil {
		t.Fatal(err)
	}
	recorder := &displaytest.CallRecorder{}
	opts := &attachOptions{
		Target:       option.Target{Type: option.TargetTypeOCILayout, Path: layout, Reference: "source"},
		Packer:       option.Packer{FileRefs: []string{fileRef}},
		artifactType: "application/example.artifact",
	}
	handlers := attachHandlers{
		status:   &displaytest.RecordingAttachStatusHandler{Recorder: recorder},
		metadata: &displaytest.RecordingAttachMetadataHandler{Recorder: recorder},
	}
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	if err := runAttach(cmd, opts, handlers); err != nil {
		t.Fatal(err)
	}
	if len(recorder.Snapshot()) == 0 {
		t.Fatal("attach handlers recorded no calls")
	}
}
