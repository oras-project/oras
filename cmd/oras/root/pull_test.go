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
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	metadatajson "oras.land/oras/cmd/oras/internal/display/metadata/json"
	"oras.land/oras/cmd/oras/internal/display/status"
	orerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

func Test_runPull_errType(t *testing.T) {
	// prepare
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// test
	opts := &pullOptions{
		Format: option.Format{
			Type: "unknown",
		},
	}
	got := runPull(cmd, opts).Error()
	want := orerrors.UnsupportedFormatTypeError(opts.Format.Type).Error()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestNewPullCleanup_NonexistentOutput(t *testing.T) {
	output := filepath.Join(t.TempDir(), "output")

	cleanup, err := newPullCleanup(output)
	if err != nil {
		t.Fatalf("newPullCleanup() error = %v", err)
	}
	if len(cleanup.existing) != 0 {
		t.Fatalf("newPullCleanup() existing = %d, want 0", len(cleanup.existing))
	}
}

func TestNewPullCleanup_WalkDirError(t *testing.T) {
	output := t.TempDir()
	subdir := filepath.Join(output, "subdir")

	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(subdir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(subdir, 0o755)
	})

	_, err := newPullCleanup(output)
	if err == nil {
		t.Fatal("newPullCleanup() error = nil, want error")
	}
}

func TestPullCleanup_CleanupRemoveError(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	path := filepath.Join(subdir, "file")

	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	cleanup := &pullCleanup{
		existing: make(map[string]struct{}),
		created:  map[string]struct{}{path: {}},
	}

	if err := os.Chmod(subdir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(subdir, 0o755)
	})

	err := cleanup.cleanup()
	if err == nil {
		t.Fatal("cleanup() error = nil, want error")
	}
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("cleanup() error = %v, want permission error", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should remain after failed cleanup: %v", err)
	}
}

func TestDoPull_TracksAbsoluteFilePath(t *testing.T) {
	ctx := context.Background()

	src := memory.New()
	dst := memory.New()

	content := []byte("test content\n")
	filePath := filepath.Join(t.TempDir(), "tool")

	layerDesc := ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    digest.FromBytes(content),
		Size:      int64(len(content)),
		Annotations: map[string]string{
			ocispec.AnnotationTitle: filePath,
		},
	}

	if err := src.Push(ctx, layerDesc, bytes.NewReader(content)); err != nil {
		t.Fatalf("Push() layer error = %v", err)
	}

	manifest := ocispec.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType: ocispec.MediaTypeImageManifest,
		Config: ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageConfig,
			Digest:    ocispec.DescriptorEmptyJSON.Digest,
			Size:      ocispec.DescriptorEmptyJSON.Size,
		},
		Layers: []ocispec.Descriptor{
			layerDesc,
		},
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	manifestDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    digest.FromBytes(manifestBytes),
		Size:      int64(len(manifestBytes)),
	}

	if err := src.Push(ctx, manifestDesc, bytes.NewReader(manifestBytes)); err != nil {
		t.Fatalf("Push() manifest error = %v", err)
	}
	if err := src.Tag(ctx, manifestDesc, "test"); err != nil {
		t.Fatalf("Tag() error = %v", err)
	}

	output := t.TempDir()

	cleanup, err := newPullCleanup(output)
	if err != nil {
		t.Fatalf("newPullCleanup() error = %v", err)
	}

	statusHandler := status.NewDiscardHandler()
	metadataHandler := metadatajson.NewPullHandler(&bytes.Buffer{}, output)

	po := &pullOptions{
		Target: option.Target{
			Reference: "test",
		},
		Output: output,
	}

	_, err = doPull(
		ctx,
		src,
		dst,
		oras.CopyOptions{},
		metadataHandler,
		statusHandler,
		po,
		cleanup,
	)
	if err != nil {
		t.Fatalf("doPull() error = %v", err)
	}

	if _, ok := cleanup.created[filePath]; !ok {
		t.Fatalf("cleanup.created does not contain %q", filePath)
	}
}
