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

package dir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWalk(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()

	// Create test structure:
	// tmpDir/
	//   file1.txt
	//   file2.txt
	//   subdir1/
	//     file3.txt
	//     nested/
	//       file4.txt
	//   subdir2/
	//     file5.txt
	//   empty/

	files := []string{
		"file1.txt",
		"file2.txt",
		"subdir1/file3.txt",
		"subdir1/nested/file4.txt",
		"subdir2/file5.txt",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	// Create empty directory
	if err := os.MkdirAll(filepath.Join(tmpDir, "empty"), 0755); err != nil {
		t.Fatalf("failed to create empty directory: %v", err)
	}

	t.Run("basic walk", func(t *testing.T) {
		node, err := Walk(tmpDir, WalkOptions{})
		if err != nil {
			t.Fatalf("Walk() error = %v", err)
		}

		if !node.IsDir {
			t.Error("root node should be a directory")
		}

		fileCount := node.FileCount()
		if fileCount != 5 {
			t.Errorf("FileCount() = %d, want 5", fileCount)
		}

		// Empty directory should be pruned by default
		dirCount := node.DirCount()
		if dirCount != 4 { // root + subdir1 + nested + subdir2
			t.Errorf("DirCount() = %d, want 4", dirCount)
		}
	})

	t.Run("walk with empty dirs", func(t *testing.T) {
		node, err := Walk(tmpDir, WalkOptions{IncludeEmpty: true})
		if err != nil {
			t.Fatalf("Walk() error = %v", err)
		}

		dirCount := node.DirCount()
		if dirCount != 5 { // root + subdir1 + nested + subdir2 + empty
			t.Errorf("DirCount() = %d, want 5", dirCount)
		}
	})

	t.Run("walk with exclusions", func(t *testing.T) {
		node, err := Walk(tmpDir, WalkOptions{
			ExcludePatterns: []string{"subdir1"},
		})
		if err != nil {
			t.Fatalf("Walk() error = %v", err)
		}

		fileCount := node.FileCount()
		if fileCount != 3 { // file1.txt, file2.txt, subdir2/file5.txt
			t.Errorf("FileCount() = %d, want 3", fileCount)
		}
	})
}

func TestNode_Methods(t *testing.T) {
	// Create a test node structure manually
	root := &Node{
		Name:  "root",
		Path:  ".",
		IsDir: true,
		Children: []*Node{
			{Name: "file1.txt", Path: "file1.txt", IsDir: false, Size: 100},
			{Name: "file2.txt", Path: "file2.txt", IsDir: false, Size: 200},
			{
				Name:  "subdir",
				Path:  "subdir",
				IsDir: true,
				Children: []*Node{
					{Name: "file3.txt", Path: "subdir/file3.txt", IsDir: false, Size: 150},
				},
			},
		},
	}

	t.Run("FileCount", func(t *testing.T) {
		if got := root.FileCount(); got != 3 {
			t.Errorf("FileCount() = %d, want 3", got)
		}
	})

	t.Run("DirCount", func(t *testing.T) {
		if got := root.DirCount(); got != 2 {
			t.Errorf("DirCount() = %d, want 2", got)
		}
	})

	t.Run("HasFiles", func(t *testing.T) {
		if !root.HasFiles() {
			t.Error("HasFiles() = false, want true")
		}
	})

	t.Run("HasDirs", func(t *testing.T) {
		if !root.HasDirs() {
			t.Error("HasDirs() = false, want true")
		}
	})

	t.Run("Files", func(t *testing.T) {
		files := root.Files()
		if len(files) != 2 {
			t.Errorf("Files() length = %d, want 2", len(files))
		}
	})

	t.Run("Dirs", func(t *testing.T) {
		dirs := root.Dirs()
		if len(dirs) != 1 {
			t.Errorf("Dirs() length = %d, want 1", len(dirs))
		}
	})
}

func TestFlattenFiles(t *testing.T) {
	root := &Node{
		Name:  "root",
		Path:  ".",
		IsDir: true,
		Children: []*Node{
			{Name: "file1.txt", Path: "file1.txt", IsDir: false},
			{
				Name:  "subdir",
				Path:  "subdir",
				IsDir: true,
				Children: []*Node{
					{Name: "file2.txt", Path: "subdir/file2.txt", IsDir: false},
					{Name: "file3.txt", Path: "subdir/file3.txt", IsDir: false},
				},
			},
		},
	}

	files := FlattenFiles(root)
	if len(files) != 3 {
		t.Errorf("FlattenFiles() length = %d, want 3", len(files))
	}

	// Check paths
	expectedPaths := map[string]bool{
		"file1.txt":        true,
		"subdir/file2.txt": true,
		"subdir/file3.txt": true,
	}
	for _, f := range files {
		if !expectedPaths[f.Path] {
			t.Errorf("unexpected file path: %s", f.Path)
		}
	}
}

func TestChunkFiles(t *testing.T) {
	nodes := []*Node{
		{Name: "1"},
		{Name: "2"},
		{Name: "3"},
		{Name: "4"},
		{Name: "5"},
	}

	tests := []struct {
		name     string
		maxSize  int
		expected int
	}{
		{"no chunking needed", 10, 1},
		{"exact split", 5, 1},
		{"two chunks", 3, 2},
		{"five chunks", 1, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := ChunkFiles(nodes, tt.maxSize)
			if len(chunks) != tt.expected {
				t.Errorf("ChunkFiles() chunks = %d, want %d", len(chunks), tt.expected)
			}
		})
	}
}

func TestWalk_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "single.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	node, err := Walk(filePath, WalkOptions{})
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	if node.IsDir {
		t.Error("node should not be a directory")
	}
	if node.Name != "single.txt" {
		t.Errorf("node.Name = %q, want %q", node.Name, "single.txt")
	}
}
