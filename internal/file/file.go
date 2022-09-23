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

package file

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/file"
)

// LoadFiles loads files and returns their descriptors.
func LoadFiles(ctx context.Context, store *file.Store, annotations map[string]map[string]string, fileRefs []string, verbose bool) ([]ocispec.Descriptor, error) {
	var files []ocispec.Descriptor
	for _, fileRef := range fileRefs {
		filename, mediaType := ParseFileReference(fileRef, "")

		// get shortest absolute path as unique name
		name := filepath.Clean(filename)
		if !filepath.IsAbs(name) {
			name = filepath.ToSlash(name)
		}

		if verbose {
			fmt.Println("Preparing", name)
		}
		file, err := store.Add(ctx, name, mediaType, filename)
		if err != nil {
			return nil, err
		}
		if value, ok := annotations[filename]; ok {
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

// PrepareManifestContent prepares the content for manifest from the file path
// or stdin.
func PrepareManifestContent(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("missing file name")
	}

	var content []byte
	var err error
	if path == "-" {
		content, err = io.ReadAll(os.Stdin)
	} else {
		content, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	return content, nil
}

// PrepareBlobContent prepares the content descriptor for blob from the file
// path or stdin. Use the input digest and size if they are provided. Will
// return error if the content is from stdin but the content digest and size
// are missing.
func PrepareBlobContent(path string, mediaType string, dgstStr string, size int64) (desc ocispec.Descriptor, rc io.ReadCloser, prepareErr error) {
	if path == "" {
		return ocispec.Descriptor{}, nil, errors.New("missing file name")
	}

	// validate digest
	var dgst digest.Digest
	if dgstStr != "" {
		var err error
		dgst, err = digest.Parse(dgstStr)
		if err != nil {
			return ocispec.Descriptor{}, nil, fmt.Errorf("invalid digest %s: %w", dgstStr, err)
		}
	}

	// prepares the content descriptor from stdin
	if path == "-" {
		// throw err if size or digest is not provided.
		if size < 0 {
			return ocispec.Descriptor{}, nil, errors.New("content size must be provided if it is read from stdin")
		}
		if dgst == "" {
			return ocispec.Descriptor{}, nil, errors.New("content digest must be provided if it is read from stdin")
		}
		return ocispec.Descriptor{
			MediaType: mediaType,
			Digest:    dgst,
			Size:      size,
		}, os.Stdin, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return ocispec.Descriptor{}, nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer func() {
		if prepareErr != nil {
			file.Close()
		}
	}()

	fi, err := file.Stat()
	if err != nil {
		return ocispec.Descriptor{}, nil, fmt.Errorf("failed to stat %s: %w", path, err)
	}
	actualSize := fi.Size()
	if size >= 0 && size != actualSize {
		return ocispec.Descriptor{}, nil, fmt.Errorf("input size %d does not match the actual content size %d", size, actualSize)
	}

	if dgst == "" {
		dgst, err = digest.FromReader(file)
		if err != nil {
			return ocispec.Descriptor{}, nil, err
		}
		if _, err = file.Seek(0, io.SeekStart); err != nil {
			return ocispec.Descriptor{}, nil, err
		}
	}

	return ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    dgst,
		Size:      actualSize,
	}, file, nil
}

// ParseMediaType parses the media type field of bytes content in json format.
func ParseMediaType(content []byte) (string, error) {
	var manifest struct {
		MediaType string `json:"mediaType"`
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		return "", errors.New("not a valid json file")
	}
	if manifest.MediaType == "" {
		return "", errors.New("media type is not recognized")
	}
	return manifest.MediaType, nil
}
