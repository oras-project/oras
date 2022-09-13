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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// PrepareManifestContent prepares the content descriptor from the file path or stdin.
func PrepareManifestContent(path string, mediaType string) (desc ocispec.Descriptor, rc io.ReadCloser, prepareErr error) {
	if path == "" {
		return ocispec.Descriptor{}, nil, fmt.Errorf("missing file name")
	}

	var file *os.File
	var err error
	if path == "-" {
		file = os.Stdin
	} else {
		file, err := os.Open(path)
		if err != nil {
			return ocispec.Descriptor{}, nil, fmt.Errorf("failed to open %s: %w", path, err)
		}
		defer func() {
			if prepareErr != nil {
				file.Close()
			}
		}()
	}

	fi, err := file.Stat()
	if err != nil {
		return ocispec.Descriptor{}, nil, fmt.Errorf("failed to stat %s: %w", path, err)
	}

	dgst, err := digest.FromReader(file)
	if err != nil {
		return ocispec.Descriptor{}, nil, err
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return ocispec.Descriptor{}, nil, err
	}

	desc = ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    dgst,
		Size:      fi.Size(),
	}
	return desc, file, nil
}

// PrepareBlobContent prepares the content descriptor from the file path or stdin.
// Use the input digest and size if they are provided. Will return error if the
// content is from stdin but the content digest and size are missing.
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

func ParseMediaType(path string) (string, error) {
	manifestByte, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestByte, &manifest); err != nil {
		return "", errors.New("not a valid json file")
	}
	if manifest["mediaType"] == nil {
		return "", errors.New("media type is not recognized")
	}
	return fmt.Sprint(manifest["mediaType"]), nil
}
