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
	"fmt"
	"os"
	"path/filepath"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras/cmd/oras/internal/fileref"
)

func loadFiles(ctx context.Context, store *file.Store, annotations map[string]map[string]string, fileRefs []string, verbose bool) ([]ocispec.Descriptor, error) {
	var files []ocispec.Descriptor
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for _, fileRef := range fileRefs {
		path, mediaType, err := fileref.Parse(fileRef, "")
		if err != nil {
			return nil, err
		}
		path, name := getPathName(path, wd)
		if verbose {
			fmt.Println("Preparing", name)
		}
		file, err := store.Add(ctx, name, mediaType, path)
		if err != nil {
			return nil, err
		}
		if value, ok := annotations[name]; ok {
			if file.Annotations == nil {
				file.Annotations = value
			} else {
				for k, v := range value {
					file.Annotations[k] = v
				}
			}
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		fmt.Println("Uploading empty artifact")
	}
	return files, nil
}

func getPathName(path string, root string) (string, string) {
	// get shortest relative path as unique name
	name := filepath.Clean(path)
	if !filepath.IsAbs(name) {
		name = filepath.ToSlash(name)
		path = filepath.Join(root, path)
	}
	return path, name
}
