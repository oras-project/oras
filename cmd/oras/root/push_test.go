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
	stderrors "errors"
	"os"
	"path/filepath"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/status"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/contentutil"
)

type recordingPushStatusHandler struct {
	status.DiscardHandler
	fileLoadingCalls []string
	onFileLoadingErr error
}

func (h *recordingPushStatusHandler) OnFileLoading(name string) error {
	h.fileLoadingCalls = append(h.fileLoadingCalls, name)
	return h.onFileLoadingErr
}

type recordingPushMetadataHandler struct {
	metadata.Discard
	copiedCalls int
	taggedCalls []string
	renderCalls int
}

func (h *recordingPushMetadataHandler) OnCopied(
	_ *option.Target,
	_ ocispec.Descriptor,
) error {
	h.copiedCalls++
	return nil
}

func (h *recordingPushMetadataHandler) OnTagged(
	_ ocispec.Descriptor,
	tag string,
) error {
	h.taggedCalls = append(h.taggedCalls, tag)
	return nil
}

func (h *recordingPushMetadataHandler) Render() error {
	h.renderCalls++
	return nil
}

func Test_runPush_errType(t *testing.T) {
	// prepare
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// test
	opts := &pushOptions{
		Format: option.Format{
			Type: "unknown",
		},
	}
	got := runPush(cmd, opts).Error()
	want := errors.UnsupportedFormatTypeError(opts.Format.Type).Error()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// Test_runPush_force exercises pushing with --force, including the extra-tag
// path that uses the TraversingTarget-wrapped destination.
func Test_runPush_force(t *testing.T) {
	// prepare a file to push
	tempDir := t.TempDir()
	fileName := "hi.txt"
	filePath := filepath.Join(tempDir, fileName)
	if err := os.WriteFile(filePath, []byte("hello world"), 0600); err != nil {
		t.Fatal(err)
	}

	// use an OCI image layout as the push destination
	layoutDir := filepath.Join(tempDir, "layout")

	cmd := pushCmd()
	cmd.SetArgs([]string{
		"--oci-layout",
		"--force",
		"--disable-path-validation",
		layoutDir + ":tag1,tag2",
		filePath,
	})
	cmd.SetContext(context.Background())
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// verify the artifact and both tags were pushed to the layout
	store, err := oci.New(layoutDir)
	if err != nil {
		t.Fatalf("failed to open pushed layout: %v", err)
	}
	for _, tag := range []string{"tag1", "tag2"} {
		if _, err := store.Resolve(context.Background(), tag); err != nil {
			t.Errorf("expected tag %q to be resolvable, got error: %v", tag, err)
		}
	}
}

func Test_pushCmd_invalidExtraTag(t *testing.T) {
	cmd := pushCmd()
	cmd.SetArgs([]string{
		"localhost:5000/hello:v1,,invalid@@tag",
		"hi.txt",
	})

	err := cmd.Execute()
	if err == nil || err.Error() != "empty extra tag" {
		t.Fatalf("got %v, want empty extra tag error", err)
	}
}

func Test_doPushCommand_fileLoadingError(t *testing.T) {
	ctx := context.Background()
	wantErr := stderrors.New("file loading failed")
	handler := &recordingPushStatusHandler{
		onFileLoadingErr: wantErr,
	}

	opts := &pushOptions{}
	opts.FileRefs = []string{"unused.txt"}

	targetCreated := false
	err := doPushCommand(
		ctx,
		opts,
		oras.PackManifestOptions{},
		nil,
		nil,
		nil,
		handler,
		&recordingPushMetadataHandler{},
		func() (oras.GraphTarget, error) {
			targetCreated = true
			return nil, stderrors.New("unexpected target creation")
		},
	)

	if !stderrors.Is(err, wantErr) {
		t.Fatalf("doPushCommand() error = %v, want %v", err, wantErr)
	}
	if targetCreated {
		t.Fatal("destination was created after file loading failed")
	}
	if len(handler.fileLoadingCalls) != 1 {
		t.Fatalf("OnFileLoading called %d times, want 1", len(handler.fileLoadingCalls))
	}
}

func Test_doPushCommand_displayHandlerCalls(t *testing.T) {
	ctx := context.Background()

	filePath := filepath.Join(t.TempDir(), "hello.txt")
	if err := os.WriteFile(filePath, []byte("hello world"), 0600); err != nil {
		t.Fatal(err)
	}

	store, err := file.New("")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	memoryStore := memory.New()
	union := contentutil.MultiReadOnlyTarget(memoryStore, store)
	destination := memory.New()

	statusHandler := &recordingPushStatusHandler{}
	metadataHandler := &recordingPushMetadataHandler{}

	opts := &pushOptions{
		Packer: option.Packer{
			FileRefs: []string{filePath},
		},
		ImageSpec: option.ImageSpec{
			PackVersion: oras.PackManifestVersion1_1,
		},
		artifactType: "application/vnd.example.test",
		extraRefs:    []string{"tag2"},
	}

	err = doPushCommand(
		ctx,
		opts,
		oras.PackManifestOptions{},
		store,
		memoryStore,
		union,
		statusHandler,
		metadataHandler,
		func() (oras.GraphTarget, error) {
			return destination, nil
		},
	)
	if err != nil {
		t.Fatalf("doPushCommand() error = %v", err)
	}

	if len(statusHandler.fileLoadingCalls) != 1 {
		t.Errorf("OnFileLoading called %d times, want 1", len(statusHandler.fileLoadingCalls))
	}
	if metadataHandler.copiedCalls != 1 {
		t.Errorf("OnCopied called %d times, want 1", metadataHandler.copiedCalls)
	}
	if len(metadataHandler.taggedCalls) != 1 || metadataHandler.taggedCalls[0] != "tag2" {
		t.Errorf("OnTagged calls = %v, want [tag2]", metadataHandler.taggedCalls)
	}
	if metadataHandler.renderCalls != 1 {
		t.Errorf("Render called %d times, want 1", metadataHandler.renderCalls)
	}
}
