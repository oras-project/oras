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

package content

import (
	"context"
	"fmt"
	"path/filepath"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/file"
)

// FileReference refers to a local file.
type FileReference struct {
	FileName  string
	MediaType string
}

// NewFileReference creates a new file reference struct.
func NewFileReference(filePath string, mediaType string) FileReference {
	return FileReference{
		FileName:  filePath,
		MediaType: mediaType,
	}
}

// LoadFiles loads file references to a file store and and returns the
// descriptors.
func LoadFiles(ctx context.Context, store *file.Store, annotations map[string]map[string]string, refs []FileReference, verbose bool) ([]ocispec.Descriptor, error) {
	files := make([]ocispec.Descriptor, len(refs))
	for i, ref := range refs {
		name := filepath.Clean(ref.FileName)
		if !filepath.IsAbs(name) {
			// convert to slash-separated path unless it is absolute path
			name = filepath.ToSlash(name)
		}
		if verbose {
			fmt.Println("Preparing", name)
		}
		desc, err := store.Add(ctx, name, ref.MediaType, ref.FileName)
		if err != nil {
			return nil, err
		}
		if annotations != nil {
			if value, ok := annotations[ref.FileName]; ok {
				if desc.Annotations == nil {
					desc.Annotations = value
				} else {
					for k, v := range value {
						desc.Annotations[k] = v
					}
				}
			}
		}
		files[i] = desc
	}
	return files, nil
}
