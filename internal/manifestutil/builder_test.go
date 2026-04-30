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

package manifestutil

import (
	"context"
	"encoding/json"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras/internal/dir"
)

// TestBuildFromNode_noAnnotationTitleOnIndexDescriptors is a regression test for
// the bug reported in https://github.com/oras-project/oras/pull/1951 where
// oras pull would fail with "open <path>: is a directory" after a recursive push.
//
// Root cause: Image Index descriptors stored in parent index manifests lists
// carried org.opencontainers.image.title = "<dirname>". oras pull sees any
// descriptor with AnnotationTitle and calls metadataHandler.OnFilePulled, which
// tries to open/create that path as a file — failing when a directory with that
// name already exists on disk.
func TestBuildFromNode_noAnnotationTitleOnIndexDescriptors(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	b := NewBuilder(store, BuilderOptions{MaxBlobsPerManifest: 1000})

	// Build a tree that produces a root Image Index with a child Image Index
	// (for "subdir") — this is the exact shape that triggers the pull bug.
	root := &dir.Node{
		Name:  "root",
		Path:  ".",
		IsDir: true,
		Children: []*dir.Node{
			{
				Name:  "subdir",
				Path:  "subdir",
				IsDir: true,
				Children: []*dir.Node{
					{Name: "file.txt", Path: "subdir/file.txt", IsDir: false},
				},
			},
		},
	}
	fileDesc := content.NewDescriptorFromBytes("application/vnd.oci.image.layer.v1.tar", []byte("hello"))
	fileDesc.Annotations = map[string]string{ocispec.AnnotationTitle: "subdir/file.txt"}
	fileDescs := map[string]ocispec.Descriptor{
		"subdir/file.txt": fileDesc,
	}

	res, err := b.BuildFromNode(ctx, root, fileDescs)
	if err != nil {
		t.Fatalf("BuildFromNode() error = %v", err)
	}
	if res.Root.MediaType != ocispec.MediaTypeImageIndex {
		t.Fatalf("root MediaType = %q, want %q", res.Root.MediaType, ocispec.MediaTypeImageIndex)
	}

	// Fetch the root index JSON from the store and inspect its manifest entries.
	rc, err := store.Fetch(ctx, res.Root)
	if err != nil {
		t.Fatalf("Fetch root index: %v", err)
	}
	defer rc.Close()

	var idx ocispec.Index
	if err := json.NewDecoder(rc).Decode(&idx); err != nil {
		t.Fatalf("decode index: %v", err)
	}

	// Every entry in the root index is a child directory (Image Index).
	// None of them should have AnnotationTitle — oras pull uses that annotation
	// to decide whether to write a file, so having it on an index descriptor
	// causes "open <dirname>: is a directory".
	for i, m := range idx.Manifests {
		if title, ok := m.Annotations[ocispec.AnnotationTitle]; ok {
			t.Errorf("manifest[%d] has AnnotationTitle=%q — this will cause oras pull to fail with \"is a directory\"", i, title)
		}
	}
}

func TestNewBuilder(t *testing.T) {
	store := memory.New()

	t.Run("default max blobs", func(t *testing.T) {
		builder := NewBuilder(store, BuilderOptions{})
		if builder.opts.MaxBlobsPerManifest != 1000 {
			t.Errorf("MaxBlobsPerManifest = %d, want 1000", builder.opts.MaxBlobsPerManifest)
		}
	})

	t.Run("custom max blobs", func(t *testing.T) {
		builder := NewBuilder(store, BuilderOptions{MaxBlobsPerManifest: 500})
		if builder.opts.MaxBlobsPerManifest != 500 {
			t.Errorf("MaxBlobsPerManifest = %d, want 500", builder.opts.MaxBlobsPerManifest)
		}
	})
}

func TestBuilder_BuildFromNode_FilesOnly(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	builder := NewBuilder(store, BuilderOptions{
		MaxBlobsPerManifest: 1000,
	})

	// Create a simple directory structure with only files
	rootNode := &dir.Node{
		Name:  "testdir",
		Path:  ".",
		IsDir: true,
		Children: []*dir.Node{
			{Name: "file1.txt", Path: "file1.txt", IsDir: false, Size: 100},
			{Name: "file2.txt", Path: "file2.txt", IsDir: false, Size: 200},
		},
	}

	// Create mock file descriptors
	fileDescs := map[string]ocispec.Descriptor{
		"file1.txt": {
			MediaType: "application/vnd.oci.image.layer.v1.tar",
			Digest:    "sha256:1111111111111111111111111111111111111111111111111111111111111111",
			Size:      100,
			Annotations: map[string]string{
				ocispec.AnnotationTitle: "file1.txt",
			},
		},
		"file2.txt": {
			MediaType: "application/vnd.oci.image.layer.v1.tar",
			Digest:    "sha256:2222222222222222222222222222222222222222222222222222222222222222",
			Size:      200,
			Annotations: map[string]string{
				ocispec.AnnotationTitle: "file2.txt",
			},
		},
	}

	result, err := builder.BuildFromNode(ctx, rootNode, fileDescs)
	if err != nil {
		t.Fatalf("BuildFromNode() error = %v", err)
	}

	if result.Root.Digest == "" {
		t.Error("result.Root.Digest should not be empty")
	}

	if result.ManifestCount != 1 {
		t.Errorf("ManifestCount = %d, want 1", result.ManifestCount)
	}

	if result.IndexCount != 0 {
		t.Errorf("IndexCount = %d, want 0", result.IndexCount)
	}
}

func TestBuilder_BuildFromNode_WithSubdirs(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	builder := NewBuilder(store, BuilderOptions{
		MaxBlobsPerManifest: 1000,
	})

	// Create a directory structure with subdirectories
	rootNode := &dir.Node{
		Name:  "testdir",
		Path:  ".",
		IsDir: true,
		Children: []*dir.Node{
			{Name: "file1.txt", Path: "file1.txt", IsDir: false, Size: 100},
			{
				Name:  "subdir",
				Path:  "subdir",
				IsDir: true,
				Children: []*dir.Node{
					{Name: "file2.txt", Path: "subdir/file2.txt", IsDir: false, Size: 200},
				},
			},
		},
	}

	// Create mock file descriptors
	fileDescs := map[string]ocispec.Descriptor{
		"file1.txt": {
			MediaType: "application/vnd.oci.image.layer.v1.tar",
			Digest:    "sha256:1111111111111111111111111111111111111111111111111111111111111111",
			Size:      100,
			Annotations: map[string]string{
				ocispec.AnnotationTitle: "file1.txt",
			},
		},
		"subdir/file2.txt": {
			MediaType: "application/vnd.oci.image.layer.v1.tar",
			Digest:    "sha256:2222222222222222222222222222222222222222222222222222222222222222",
			Size:      200,
			Annotations: map[string]string{
				ocispec.AnnotationTitle: "file2.txt",
			},
		},
	}

	result, err := builder.BuildFromNode(ctx, rootNode, fileDescs)
	if err != nil {
		t.Fatalf("BuildFromNode() error = %v", err)
	}

	if result.Root.Digest == "" {
		t.Error("result.Root.Digest should not be empty")
	}

	// Should have index at root (1) + manifest for root files (1) + manifest for subdir (1)
	if result.ManifestCount != 2 {
		t.Errorf("ManifestCount = %d, want 2", result.ManifestCount)
	}

	if result.IndexCount != 1 {
		t.Errorf("IndexCount = %d, want 1", result.IndexCount)
	}
}

func TestBuilder_BuildFromNode_EmptyDir(t *testing.T) {
	ctx := context.Background()
	store := memory.New()

	t.Run("without preserve empty", func(t *testing.T) {
		builder := NewBuilder(store, BuilderOptions{
			PreserveEmptyDirs: false,
		})

		rootNode := &dir.Node{
			Name:     "emptydir",
			Path:     ".",
			IsDir:    true,
			Children: []*dir.Node{},
		}

		result, err := builder.BuildFromNode(ctx, rootNode, map[string]ocispec.Descriptor{})
		if err != nil {
			t.Fatalf("BuildFromNode() error = %v", err)
		}

		if result.Root.Digest != "" {
			t.Error("empty dir without preserve should have empty digest")
		}
	})

	t.Run("with preserve empty", func(t *testing.T) {
		builder := NewBuilder(store, BuilderOptions{
			PreserveEmptyDirs: true,
		})

		rootNode := &dir.Node{
			Name:     "emptydir",
			Path:     ".",
			IsDir:    true,
			Children: []*dir.Node{},
		}

		result, err := builder.BuildFromNode(ctx, rootNode, map[string]ocispec.Descriptor{})
		if err != nil {
			t.Fatalf("BuildFromNode() error = %v", err)
		}

		if result.Root.Digest == "" {
			t.Error("empty dir with preserve should have non-empty digest")
		}
	})
}

func TestChunkDescriptors(t *testing.T) {
	descs := []ocispec.Descriptor{
		{Digest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		{Digest: "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
		{Digest: "sha256:3333333333333333333333333333333333333333333333333333333333333333"},
		{Digest: "sha256:4444444444444444444444444444444444444444444444444444444444444444"},
		{Digest: "sha256:5555555555555555555555555555555555555555555555555555555555555555"},
	}

	tests := []struct {
		name          string
		maxSize       int
		expectedCount int
	}{
		{"no chunking", 10, 1},
		{"exact fit", 5, 1},
		{"two chunks", 3, 2},
		{"three chunks", 2, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunkDescriptors(descs, tt.maxSize)
			if len(chunks) != tt.expectedCount {
				t.Errorf("chunkDescriptors() = %d chunks, want %d", len(chunks), tt.expectedCount)
			}
		})
	}
}
