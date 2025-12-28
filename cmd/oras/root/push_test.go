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
	"reflect"
	"testing"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	oras "github.com/oras-project/oras-go/v3"
	"github.com/oras-project/oras-go/v3/content/file"
	"github.com/oras-project/oras-go/v3/content/memory"
	"github.com/oras-project/oras-go/v3/content/oci"
	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/display/displaytest"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/contentutil"
)

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

func Test_pushCmd_invalidOCILayoutReference(t *testing.T) {
	cmd := pushCmd()
	cmd.SetArgs([]string{
		"--oci-layout",
		filepath.Join(t.TempDir(), "layout") + ":BAD TAG",
		"hi.txt",
	})

	err := cmd.Execute()
	if err == nil || err.Error() != `invalid reference: invalid tag "BAD TAG"` {
		t.Fatalf("got error %v, want invalid tag reference error", err)
	}
}

func Test_pushCmd_invalidOCILayoutReferenceWithSlash(t *testing.T) {
	cmd := pushCmd()
	cmd.SetArgs([]string{
		"--oci-layout",
		filepath.Join(t.TempDir(), "layout") + ":foo/bar",
		"hi.txt",
	})

	err := cmd.Execute()
	if err == nil || err.Error() != `invalid reference: invalid tag "foo/bar"` {
		t.Fatalf("got error %v, want invalid tag reference error", err)
	}
}

func Test_doPushCommand_fileLoadingError(t *testing.T) {
	ctx := context.Background()
	wantErr := stderrors.New("file loading failed")
	recorder := &displaytest.CallRecorder{}
	statusHandler := &displaytest.RecordingPushStatusHandler{
		Recorder:       recorder,
		FileLoadingErr: wantErr,
	}

	opts := &pushOptions{}
	opts.FileRefs = []string{"unused.txt"}

	targetCreated := false
	err := doPushCommand(
		ctx,
		opts,
		oras.PackManifestOptions{},
		nil, // store is never used: OnFileLoading fails before addFile
		nil,
		nil,
		statusHandler,
		&displaytest.RecordingPushMetadataHandler{Recorder: recorder},
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
	got := recorder.Snapshot()
	if !reflect.DeepEqual(got, []string{"OnFileLoading"}) {
		t.Fatalf("handler calls = %v, want [OnFileLoading]", got)
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

	recorder := &displaytest.CallRecorder{}
	statusHandler := &displaytest.RecordingPushStatusHandler{Recorder: recorder}
	metadataHandler := &displaytest.RecordingPushMetadataHandler{Recorder: recorder}

	opts := &pushOptions{}
	opts.FileRefs = []string{filePath}
	opts.Reference = "tag1"
	opts.PackVersion = oras.PackManifestVersion1_1
	opts.artifactType = "application/example"
	opts.extraRefs = []string{"tag2"}

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

	wantCalls := []string{"OnFileLoading", "OnCopied", "OnTagged:tag2", "Render"}
	got := recorder.Snapshot()
	if !reflect.DeepEqual(got, wantCalls) {
		t.Errorf("handler calls = %v, want %v", got, wantCalls)
	}
}

func Test_doPushCommand_onCopiedError(t *testing.T) {
	ctx := context.Background()
	recorder := &displaytest.CallRecorder{}
	wantErr := stderrors.New("OnCopied failed")
	metadataHandler := &displaytest.RecordingPushMetadataHandler{
		Recorder:  recorder,
		CopiedErr: wantErr,
	}

	store, err := file.New("")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	memoryStore := memory.New()
	union := contentutil.MultiReadOnlyTarget(memoryStore, store)

	filePath := filepath.Join(t.TempDir(), "hello.txt")
	if err := os.WriteFile(filePath, []byte("hello world"), 0600); err != nil {
		t.Fatal(err)
	}

	exportPath := filepath.Join(t.TempDir(), "manifest.json")
	opts := &pushOptions{}
	opts.FileRefs = []string{filePath}
	opts.Reference = "tag1"
	opts.PackVersion = oras.PackManifestVersion1_1
	opts.artifactType = "application/example"
	opts.extraRefs = []string{"tag2"}
	opts.ManifestExportPath = exportPath

	statusHandler := &displaytest.RecordingPushStatusHandler{Recorder: recorder}
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
			return memory.New(), nil
		},
	)
	if !stderrors.Is(err, wantErr) {
		t.Fatalf("doPushCommand() error = %v, want %v", err, wantErr)
	}
	if _, err := os.Stat(exportPath); !stderrors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected manifest export file not to exist, stat error = %v", err)
	}

	wantCalls := []string{"OnFileLoading", "OnCopied"}
	got := recorder.Snapshot()
	if !reflect.DeepEqual(got, wantCalls) {
		t.Errorf("handler calls = %v, want %v", got, wantCalls)
	}
}

func Test_doPushCommand_renderError(t *testing.T) {
	ctx := context.Background()
	recorder := &displaytest.CallRecorder{}
	wantErr := stderrors.New("Render failed")
	metadataHandler := &displaytest.RecordingPushMetadataHandler{
		Recorder:  recorder,
		RenderErr: wantErr,
	}

	store, err := file.New("")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	memoryStore := memory.New()
	union := contentutil.MultiReadOnlyTarget(memoryStore, store)

	filePath := filepath.Join(t.TempDir(), "hello.txt")
	if err := os.WriteFile(filePath, []byte("hello world"), 0600); err != nil {
		t.Fatal(err)
	}

	exportPath := filepath.Join(t.TempDir(), "manifest.json")
	opts := &pushOptions{}
	opts.FileRefs = []string{filePath}
	opts.Reference = "tag1"
	opts.PackVersion = oras.PackManifestVersion1_1
	opts.artifactType = "application/example"
	opts.extraRefs = []string{"tag2"}
	opts.ManifestExportPath = exportPath

	statusHandler := &displaytest.RecordingPushStatusHandler{Recorder: recorder}
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
			return memory.New(), nil
		},
	)
	if !stderrors.Is(err, wantErr) {
		t.Fatalf("doPushCommand() error = %v, want %v", err, wantErr)
	}
	if _, err := os.Stat(exportPath); !stderrors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected manifest export file not to exist, stat error = %v", err)
	}

	wantCalls := []string{"OnFileLoading", "OnCopied", "OnTagged:tag2", "Render"}
	got := recorder.Snapshot()
	if !reflect.DeepEqual(got, wantCalls) {
		t.Errorf("handler calls = %v, want %v", got, wantCalls)
	}
}

func Test_runPush_reproducible(t *testing.T) {
	ctx := context.Background()
	t.Setenv("SOURCE_DATE_EPOCH", "1700000000")

	tmpDir := t.TempDir()
	dirPath := filepath.Join(tmpDir, "srcdir")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatal(err)
	}
	fileName := "hi.txt"
	filePath := filepath.Join(dirPath, fileName)
	if err := os.WriteFile(filePath, []byte("hello world"), 0600); err != nil {
		t.Fatal(err)
	}

	// Push to layout1 with a fresh mtime on the directory.
	now1 := time.Now()
	if err := os.Chtimes(dirPath, now1, now1); err != nil {
		t.Fatal(err)
	}
	cmd := pushCmd()
	cmd.SetArgs([]string{
		"--oci-layout",
		"--disable-path-validation",
		filepath.Join(tmpDir, "layout1") + ":tag1",
		dirPath,
	})
	cmd.SetContext(ctx)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Push to layout2 with a fresh mtime on the directory.
	now2 := now1.Add(time.Hour)
	if err := os.Chtimes(dirPath, now2, now2); err != nil {
		t.Fatal(err)
	}
	cmd2 := pushCmd()
	cmd2.SetArgs([]string{
		"--oci-layout",
		"--disable-path-validation",
		filepath.Join(tmpDir, "layout2") + ":tag1",
		dirPath,
	})
	cmd2.SetContext(ctx)
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// digests must be byte-identical
	digests := map[string]string{}
	for _, layoutDir := range []string{filepath.Join(tmpDir, "layout1"), filepath.Join(tmpDir, "layout2")} {
		store, err := oci.New(layoutDir)
		if err != nil {
			t.Fatalf("failed to open layout %q: %v", layoutDir, err)
		}
		desc, err := store.Resolve(ctx, "tag1")
		if err != nil {
			t.Fatalf("failed to resolve tag1 in %q: %v", layoutDir, err)
		}
		digests[layoutDir] = desc.Digest.String()
	}
	if digests[filepath.Join(tmpDir, "layout1")] != digests[filepath.Join(tmpDir, "layout2")] {
		t.Fatalf("digests differ across runs: %v", digests)
	}
}

func Test_runPush_reproducible_unstableSDE(t *testing.T) {
	ctx := context.Background()
	t.Setenv("SOURCE_DATE_EPOCH", "1700000001")

	tmpDir := t.TempDir()
	fileName := "hi.txt"
	filePath := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(filePath, []byte("hello world"), 0600); err != nil {
		t.Fatal(err)
	}

	cmd := pushCmd()
	cmd.SetArgs([]string{
		"--oci-layout",
		tmpDir + "/layout:tag1",
		"--disable-path-validation",
		filePath,
	})
	cmd.SetContext(ctx)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store, err := oci.New(tmpDir + "/layout")
	if err != nil {
		t.Fatalf("failed to open layout: %v", err)
	}
	desc, err := store.Resolve(ctx, "tag1")
	if err != nil {
		t.Fatalf("failed to resolve tag1: %v", err)
	}
	wantCreated := "2023-11-14T22:13:21Z"
	if got := desc.Annotations[ocispec.AnnotationCreated]; got != wantCreated {
		t.Fatalf("unexpected created annotation: got %q, want %q", got, wantCreated)
	}
}

func Test_runPushRecursive_errType(t *testing.T) {
	// prepare
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// test
	opts := &pushOptions{
		Format: option.Format{
			Type: "unknown",
		},
		Recursive: option.Recursive{
			Recursive:           true,
			MaxBlobsPerManifest: 1000,
		},
	}
	opts.FileRefs = []string{tmpDir}

	got := runPushRecursive(cmd, opts).Error()
	want := errors.UnsupportedFormatTypeError(opts.Format.Type).Error()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func Test_runPushRecursive_emptyDir(t *testing.T) {
	// prepare
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	tmpDir := t.TempDir()

	// test with empty directory and no preserve-empty-dirs
	opts := &pushOptions{
		Recursive: option.Recursive{
			Recursive:           true,
			MaxBlobsPerManifest: 1000,
			PreserveEmptyDirs:   false,
		},
	}
	opts.FileRefs = []string{tmpDir}

	err := runPushRecursive(cmd, opts)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
	if err.Error() != "directory is empty; nothing to push" {
		t.Fatalf("got error %q, want %q", err.Error(), "directory is empty; nothing to push")
	}
}
