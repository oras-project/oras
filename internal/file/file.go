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
	"fmt"
	"io"
	"os"

	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// PrepareContent prepares the content descriptor from the file path or stdin.
// Use the input digest and size if they are provided.
func PrepareContent(path string, mediaType string, dgst digest.Digest, size int64) (ocispec.Descriptor, io.ReadCloser, error) {
	if path == "" {
		return ocispec.Descriptor{}, nil, fmt.Errorf("missing file name")
	}

	var file *os.File
	var err error
	if path == "-" {
		file = os.Stdin
	} else {
		file, err = os.Open(path)
		if err != nil {
			return ocispec.Descriptor{}, nil, fmt.Errorf("failed to open %s: %w", path, err)
		}
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

	if size <= 0 {
		fi, err := file.Stat()
		if err != nil {
			return ocispec.Descriptor{}, nil, fmt.Errorf("failed to stat %s: %w", path, err)
		}
		size = fi.Size()
	}

	return ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    dgst,
		Size:      size,
	}, file, nil
}
