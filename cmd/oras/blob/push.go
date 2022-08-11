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

package blob

import (
	"context"
	"fmt"
	"io"
	"os"

	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/option"
)

type pushBlobOptions struct {
	option.Common
	option.Remote

	FileRef   string
	targetRef string
}

func pushBlob(opts pushBlobOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}

	// prepare blob content
	desc, fp, err := packBlob(ctx, &opts)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := fp.Close()
		if err == nil {
			err = closeErr
		}
	}()

	// push blob
	if err = repo.Push(ctx, desc, fp); err != nil {
		return err
	}

	fmt.Println("Pushed", opts.targetRef)
	fmt.Println("Digest:", desc.Digest)

	return nil
}

func packBlob(ctx context.Context, opts *pushBlobOptions) (ocispec.Descriptor, *os.File, error) {
	filename := opts.FileRef
	if filename == "" {
		return ocispec.Descriptor{}, nil, fmt.Errorf("missing file name")
	}

	fp, err := os.Open(filename)
	if err != nil {
		return ocispec.Descriptor{}, nil, fmt.Errorf("failed to open %s: %w", filename, err)
	}

	fi, err := os.Stat(filename)
	if err != nil {
		return ocispec.Descriptor{}, nil, fmt.Errorf("failed to stat %s: %w", filename, err)
	}

	dgst, err := digest.FromReader(fp)
	if err != nil {
		return ocispec.Descriptor{}, nil, err
	}

	if _, err = fp.Seek(0, io.SeekStart); err != nil {
		return ocispec.Descriptor{}, nil, err
	}

	desc := ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    dgst,
		Size:      fi.Size(),
	}
	return desc, fp, nil
}
