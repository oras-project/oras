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

package model

import (
	"path/filepath"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/file"
)

// File records pulled files.
type File struct {
	// Path is the absolute path of the pulled file.
	Path string
	Descriptor
}

// NewFile creates a new file metadata.
func NewFile(name string, outputDir string, desc ocispec.Descriptor, descPath string) File {
	path := name
	if !filepath.IsAbs(name) {
		// ignore error since it's successfully written to file store
		path, _ = filepath.Abs(filepath.Join(outputDir, name))
	}
	if desc.Annotations[file.AnnotationUnpack] == "true" {
		path += string(filepath.Separator)
	}
	return File{
		Path:       path,
		Descriptor: FromDescriptor(descPath, desc),
	}
}

type pull struct {
	DigestReference
	Files []File `json:"Files"`
}

// NewPull creates a new metadata struct for pull command.
func NewPull(digestReference string, files []File) any {
	return pull{
		DigestReference: DigestReference{
			Ref: digestReference,
		},
		Files: files,
	}
}
