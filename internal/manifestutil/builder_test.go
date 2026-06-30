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
// Root cause: descriptors stored in an index's manifests list had
// org.opencontainers.image.title = "<dirname>". oras pull's PostCopy calls
// content.Successors on each copied descriptor; for any successor that has
// AnnotationTitle it calls metadataHandler.OnFilePulled, which tries to
// open/create that path as a file — failing with "is a directory" when a
// directory with that name already exists on disk.
//
// Two shapes trigger the bug:
//  1. Root has only subdirectories (child Image Index descriptors carry the title)
//  2. Root has both files and subdirectories (chunk manifest descriptors stored
//     inside the index carry the directory name as AnnotationTitle)
func TestBuildFromNode_noAnnotationTitleOnIndexDescriptors(t *testing.T) {
	t.Run("only subdirs at root", func(t *testing.T) {
		ctx := context.Background()
		store := memory.New()
		b := NewBuilder(store, BuilderOptions{MaxBlobsPerManifest: 1000})

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
		fileDescs := map[string]ocispec.Descriptor{"subdir/file.txt": fileDesc}

		assertNoAnnotationTitleInAllIndexes(t, ctx, store, b, root, fileDescs)
	})

	t.Run("files and subdirs at root", func(t *testing.T) {
		ctx := context.Background()
		store := memory.New()
		b := NewBuilder(store, BuilderOptions{MaxBlobsPerManifest: 1000})

		// This shape triggers the second variant of the bug: the root has both
		// files and a subdirectory, so buildNode creates a chunk manifest for
		// the root files. If that chunk manifest descriptor has AnnotationTitle
		// = root.Name, oras pull calls OnFilePulled(root.Name) which fails when
		// root.Name already exists as a directory.
		//
		// Similarly, "subdir" has files AND a nested subdirectory, so it also
		// produces a chunk manifest with AnnotationTitle = "subdir" — which
		// fails when the subdir directory exists on disk.
		root := &dir.Node{
			Name:  "mydir",
			Path:  ".",
			IsDir: true,
			Children: []*dir.Node{
				{Name: "root.txt", Path: "root.txt", IsDir: false},
				{
					Name:  "subdir",
					Path:  "subdir",
					IsDir: true,
					Children: []*dir.Node{
						{Name: "sub.txt", Path: "subdir/sub.txt", IsDir: false},
						{
							Name:  "nested",
							Path:  "subdir/nested",
							IsDir: true,
							Children: []*dir.Node{
								{Name: "deep.txt", Path: "subdir/nested/deep.txt", IsDir: false},
							},
						},
					},
				},
			},
		}
		mkDesc := func(path, data string) ocispec.Descriptor {
			d := content.NewDescriptorFromBytes("application/octet-stream", []byte(data))
			d.Annotations = map[string]string{ocispec.AnnotationTitle: path}
			return d
		}
		fileDescs := map[string]ocispec.Descriptor{
			"root.txt":              mkDesc("root.txt", "root"),
			"subdir/sub.txt":        mkDesc("subdir/sub.txt", "sub"),
			"subdir/nested/deep.txt": mkDesc("subdir/nested/deep.txt", "deep"),
		}

		assertNoAnnotationTitleInAllIndexes(t, ctx, store, b, root, fileDescs)
	})
}

// assertNoAnnotationTitleInAllIndexes builds from node and recursively checks
// every Image Index in the store for manifest entries that have AnnotationTitle.
func assertNoAnnotationTitleInAllIndexes(t *testing.T, ctx context.Context, store interface {
	content.Pusher
	content.Fetcher
}, b *Builder, root *dir.Node, fileDescs map[string]ocispec.Descriptor) {
	t.Helper()

	res, err := b.BuildFromNode(ctx, root, fileDescs)
	if err != nil {
		t.Fatalf("BuildFromNode() error = %v", err)
	}
	if res.Root.MediaType != ocispec.MediaTypeImageIndex {
		t.Fatalf("root MediaType = %q, want %q", res.Root.MediaType, ocispec.MediaTypeImageIndex)
	}

	// Recursively walk every index and assert no manifest entry has AnnotationTitle.
	var checkIndex func(desc ocispec.Descriptor, label string)
	checkIndex = func(desc ocispec.Descriptor, label string) {
		rc, err := store.Fetch(ctx, desc)
		if err != nil {
			t.Fatalf("Fetch %s: %v", label, err)
		}
		defer rc.Close()

		var idx ocispec.Index
		if err := json.NewDecoder(rc).Decode(&idx); err != nil {
			t.Fatalf("decode %s: %v", label, err)
		}

		for i, m := range idx.Manifests {
			if title, ok := m.Annotations[ocispec.AnnotationTitle]; ok {
				t.Errorf("%s manifest[%d] has AnnotationTitle=%q — "+
					"oras pull will call OnFilePulled(%q) and fail with \"is a directory\"",
					label, i, title, title)
			}
			// Recurse into child indexes.
			if m.MediaType == ocispec.MediaTypeImageIndex {
				checkIndex(m, label+"/manifest["+string(rune('0'+i))+"]")
			}
		}
	}
	checkIndex(res.Root, "root")
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
