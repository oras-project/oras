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
	"errors"
	"fmt"
	"io"
	"strconv"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras/internal/dir"
)

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

// failingPusher fails pushes whose media type matches failOn with err.
type failingPusher struct {
	failOn string
	err    error
}

func (p *failingPusher) Push(_ context.Context, desc ocispec.Descriptor, r io.Reader) error {
	if _, err := io.ReadAll(r); err != nil {
		return err
	}
	if desc.MediaType == p.failOn {
		return p.err
	}
	return nil
}

func testFileDesc(i int) ocispec.Descriptor {
	return ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.layer.v1.tar",
		Digest:    content.NewDescriptorFromBytes("", []byte(strconv.Itoa(i))).Digest,
		Size:      1,
	}
}

// nestedTestNode returns a root dir containing a file and a subdir with a file.
func nestedTestNode() (*dir.Node, map[string]ocispec.Descriptor) {
	root := &dir.Node{
		Name:  "root",
		Path:  ".",
		IsDir: true,
		Children: []*dir.Node{
			{Name: "a.txt", Path: "a.txt"},
			{Name: "sub", Path: "sub", IsDir: true, Children: []*dir.Node{
				{Name: "b.txt", Path: "sub/b.txt"},
			}},
		},
	}
	return root, map[string]ocispec.Descriptor{
		"a.txt":     testFileDesc(1),
		"sub/b.txt": testFileDesc(2),
	}
}

func TestBuilder_BuildFromNode_PushErrors(t *testing.T) {
	pushErr := errors.New("push failed")
	for _, mediaType := range []string{
		ocispec.MediaTypeImageConfig,
		ocispec.MediaTypeImageManifest,
		ocispec.MediaTypeImageIndex,
	} {
		t.Run(mediaType, func(t *testing.T) {
			root, fileDescs := nestedTestNode()
			builder := NewBuilder(&failingPusher{failOn: mediaType, err: pushErr}, BuilderOptions{})
			if _, err := builder.BuildFromNode(context.Background(), root, fileDescs); !errors.Is(err, pushErr) {
				t.Fatalf("BuildFromNode() error = %v, want %v", err, pushErr)
			}
		})
	}
}

func TestBuilder_BuildFromNode_ChunkPushError(t *testing.T) {
	pushErr := errors.New("push failed")
	root := &dir.Node{Name: "root", Path: ".", IsDir: true}
	fileDescs := map[string]ocispec.Descriptor{}
	for i := range 3 {
		name := fmt.Sprintf("f%d", i)
		root.Children = append(root.Children, &dir.Node{Name: name, Path: name})
		fileDescs[name] = testFileDesc(i)
	}
	builder := NewBuilder(&failingPusher{failOn: ocispec.MediaTypeImageManifest, err: pushErr}, BuilderOptions{MaxBlobsPerManifest: 1})
	if _, err := builder.BuildFromNode(context.Background(), root, fileDescs); !errors.Is(err, pushErr) {
		t.Fatalf("BuildFromNode() error = %v, want %v", err, pushErr)
	}
}

func TestBuilder_BuildFromNode_AlreadyExists(t *testing.T) {
	for _, mediaType := range []string{
		ocispec.MediaTypeImageConfig,
		ocispec.MediaTypeImageManifest,
		ocispec.MediaTypeImageIndex,
	} {
		t.Run(mediaType, func(t *testing.T) {
			root, fileDescs := nestedTestNode()
			builder := NewBuilder(&failingPusher{failOn: mediaType, err: errdef.ErrAlreadyExists}, BuilderOptions{})
			if _, err := builder.BuildFromNode(context.Background(), root, fileDescs); err != nil {
				t.Fatalf("BuildFromNode() error = %v", err)
			}
		})
	}
}

func TestBuilder_BuildFromNode_ArtifactType(t *testing.T) {
	artifactType := "application/vnd.example"
	root, fileDescs := nestedTestNode()
	builder := NewBuilder(memory.New(), BuilderOptions{ArtifactType: artifactType})
	result, err := builder.BuildFromNode(context.Background(), root, fileDescs)
	if err != nil {
		t.Fatalf("BuildFromNode() error = %v", err)
	}
	if result.Root.ArtifactType != artifactType {
		t.Errorf("root ArtifactType = %q, want %q", result.Root.ArtifactType, artifactType)
	}
	if result.IndexCount != 1 || result.ManifestCount != 2 {
		t.Errorf("IndexCount = %d, ManifestCount = %d, want 1, 2", result.IndexCount, result.ManifestCount)
	}
}

func TestBuilder_BuildFromNode_FileNode(t *testing.T) {
	builder := NewBuilder(memory.New(), BuilderOptions{})
	want := testFileDesc(1)

	result, err := builder.BuildFromNode(context.Background(), &dir.Node{Name: "a", Path: "a"}, map[string]ocispec.Descriptor{"a": want})
	if err != nil {
		t.Fatalf("BuildFromNode() error = %v", err)
	}
	if result.Root.Digest != want.Digest {
		t.Errorf("Root = %v, want %v", result.Root.Digest, want.Digest)
	}

	result, err = builder.BuildFromNode(context.Background(), &dir.Node{Name: "missing", Path: "missing"}, nil)
	if err != nil {
		t.Fatalf("BuildFromNode() error = %v", err)
	}
	if result.Root.Digest != "" {
		t.Errorf("Root = %v, want empty descriptor", result.Root.Digest)
	}
}

func TestBuilder_BuildFromNode_SkipsEmptySubdir(t *testing.T) {
	root := &dir.Node{
		Name:  "root",
		Path:  ".",
		IsDir: true,
		Children: []*dir.Node{
			{Name: "a.txt", Path: "a.txt"},
			{Name: "empty", Path: "empty", IsDir: true},
		},
	}
	builder := NewBuilder(memory.New(), BuilderOptions{})
	result, err := builder.BuildFromNode(context.Background(), root, map[string]ocispec.Descriptor{"a.txt": testFileDesc(1)})
	if err != nil {
		t.Fatalf("BuildFromNode() error = %v", err)
	}
	if result.Root.MediaType != ocispec.MediaTypeImageManifest || result.IndexCount != 0 {
		t.Errorf("got root %s with %d indexes, want a single manifest", result.Root.MediaType, result.IndexCount)
	}
}

func TestBuilder_BuildFromNode_ChunkIndexAnnotation(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	root := &dir.Node{Name: "root", Path: ".", IsDir: true}
	fileDescs := map[string]ocispec.Descriptor{}
	for i := range 12 {
		name := fmt.Sprintf("f%02d", i)
		root.Children = append(root.Children, &dir.Node{Name: name, Path: name})
		fileDescs[name] = testFileDesc(i)
	}

	builder := NewBuilder(store, BuilderOptions{MaxBlobsPerManifest: 1})
	result, err := builder.BuildFromNode(ctx, root, fileDescs)
	if err != nil {
		t.Fatalf("BuildFromNode() error = %v", err)
	}

	indexBytes, err := content.FetchAll(ctx, store, result.Root)
	if err != nil {
		t.Fatal(err)
	}
	var index ocispec.Index
	if err := json.Unmarshal(indexBytes, &index); err != nil {
		t.Fatal(err)
	}
	if len(index.Manifests) != 12 {
		t.Fatalf("got %d manifests, want 12", len(index.Manifests))
	}
	for i, m := range index.Manifests {
		if got, want := m.Annotations["org.oras.content.chunk.index"], strconv.Itoa(i); got != want {
			t.Errorf("manifest %d chunk index = %q, want %q", i, got, want)
		}
	}
}
