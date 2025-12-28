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

// Package dir provides utilities for directory traversal and tree building.
package dir

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Node represents a file or directory in the tree structure.
type Node struct {
	// Name is the base name of the file or directory.
	Name string
	// Path is the full path relative to the root.
	Path string
	// AbsPath is the absolute path on the filesystem.
	AbsPath string
	// IsDir indicates whether this node is a directory.
	IsDir bool
	// Size is the file size in bytes (0 for directories).
	Size int64
	// Children contains child nodes (only for directories).
	Children []*Node
	// Parent points to the parent node (nil for root).
	Parent *Node
}

// WalkOptions configures the directory walking behavior.
type WalkOptions struct {
	// FollowSymlinks follows symbolic links when walking.
	FollowSymlinks bool
	// IncludeEmpty includes empty directories in the tree.
	IncludeEmpty bool
	// ExcludePatterns are glob patterns for files/directories to exclude.
	ExcludePatterns []string
}

// FileCount returns the total number of files in this node and its descendants.
func (n *Node) FileCount() int {
	if !n.IsDir {
		return 1
	}
	count := 0
	for _, child := range n.Children {
		count += child.FileCount()
	}
	return count
}

// DirCount returns the total number of directories in this node and its descendants.
func (n *Node) DirCount() int {
	if !n.IsDir {
		return 0
	}
	count := 1
	for _, child := range n.Children {
		count += child.DirCount()
	}
	return count
}

// HasFiles returns true if this directory contains any files (not just subdirectories).
func (n *Node) HasFiles() bool {
	if !n.IsDir {
		return false
	}
	for _, child := range n.Children {
		if !child.IsDir {
			return true
		}
	}
	return false
}

// HasDirs returns true if this directory contains any subdirectories.
func (n *Node) HasDirs() bool {
	if !n.IsDir {
		return false
	}
	for _, child := range n.Children {
		if child.IsDir {
			return true
		}
	}
	return false
}

// Files returns all file children (non-directories) of this node.
func (n *Node) Files() []*Node {
	var files []*Node
	for _, child := range n.Children {
		if !child.IsDir {
			files = append(files, child)
		}
	}
	return files
}

// Dirs returns all directory children of this node.
func (n *Node) Dirs() []*Node {
	var dirs []*Node
	for _, child := range n.Children {
		if child.IsDir {
			dirs = append(dirs, child)
		}
	}
	return dirs
}

// Walk traverses the directory tree starting from root and builds a Node tree.
// The root path should be a directory.
func Walk(root string, opts WalkOptions) (*Node, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		// Single file, return as a node
		return &Node{
			Name:    filepath.Base(absRoot),
			Path:    filepath.Base(absRoot),
			AbsPath: absRoot,
			IsDir:   false,
			Size:    info.Size(),
		}, nil
	}

	rootNode := &Node{
		Name:    filepath.Base(absRoot),
		Path:    ".",
		AbsPath: absRoot,
		IsDir:   true,
	}

	err = walkDir(absRoot, ".", rootNode, opts)
	if err != nil {
		return nil, err
	}

	// Remove empty directories if not preserving them
	if !opts.IncludeEmpty {
		pruneEmpty(rootNode)
	}

	return rootNode, nil
}

// walkDir recursively walks a directory and builds the tree.
func walkDir(absPath, relPath string, parent *Node, opts WalkOptions) error {
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return err
	}

	// Sort entries for consistent ordering
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		name := entry.Name()
		childRelPath := filepath.Join(relPath, name)
		childAbsPath := filepath.Join(absPath, name)

		// Check exclusion patterns
		if shouldExclude(name, childRelPath, opts.ExcludePatterns) {
			continue
		}

		info, err := getFileInfo(entry, childAbsPath, opts.FollowSymlinks)
		if err != nil {
			// Skip files we can't stat (e.g., broken symlinks)
			continue
		}

		child := &Node{
			Name:    name,
			Path:    filepath.ToSlash(childRelPath),
			AbsPath: childAbsPath,
			IsDir:   info.IsDir(),
			Size:    info.Size(),
			Parent:  parent,
		}

		if child.IsDir {
			if err := walkDir(childAbsPath, childRelPath, child, opts); err != nil {
				return err
			}
		}

		parent.Children = append(parent.Children, child)
	}

	return nil
}

// getFileInfo gets file info, optionally following symlinks.
func getFileInfo(entry fs.DirEntry, path string, followSymlinks bool) (fs.FileInfo, error) {
	if entry.Type()&fs.ModeSymlink != 0 && followSymlinks {
		return os.Stat(path)
	}
	return entry.Info()
}

// shouldExclude checks if a path should be excluded based on patterns.
func shouldExclude(name, relPath string, patterns []string) bool {
	for _, pattern := range patterns {
		// Check against name
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
		// Check against relative path
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
	}
	return false
}

// pruneEmpty removes empty directories from the tree.
func pruneEmpty(node *Node) bool {
	if !node.IsDir {
		return false
	}

	// Recursively prune children
	var nonEmpty []*Node
	for _, child := range node.Children {
		if child.IsDir {
			if !pruneEmpty(child) {
				nonEmpty = append(nonEmpty, child)
			}
		} else {
			nonEmpty = append(nonEmpty, child)
		}
	}
	node.Children = nonEmpty

	// Return true if this directory is now empty
	return len(node.Children) == 0
}

// FlattenFiles returns all file nodes in the tree as a flat slice.
func FlattenFiles(root *Node) []*Node {
	var files []*Node
	flattenFilesRecursive(root, &files)
	return files
}

func flattenFilesRecursive(node *Node, files *[]*Node) {
	if !node.IsDir {
		*files = append(*files, node)
		return
	}
	for _, child := range node.Children {
		flattenFilesRecursive(child, files)
	}
}

// ChunkFiles splits a list of nodes into chunks of at most maxSize.
func ChunkFiles(nodes []*Node, maxSize int) [][]*Node {
	if maxSize <= 0 || len(nodes) <= maxSize {
		return [][]*Node{nodes}
	}

	var chunks [][]*Node
	for i := 0; i < len(nodes); i += maxSize {
		end := i + maxSize
		if end > len(nodes) {
			end = len(nodes)
		}
		chunks = append(chunks, nodes[i:end])
	}
	return chunks
}
