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
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2/errdef"

	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/option"
)

type fakeBlobDeleter struct {
	resolveErr error
}

func (f *fakeBlobDeleter) Resolve(context.Context, string) (ocispec.Descriptor, error) {
	return ocispec.Descriptor{}, f.resolveErr
}

func (f *fakeBlobDeleter) Delete(context.Context, ocispec.Descriptor) error {
	return nil
}

type recordingBlobDeleteHandler struct {
	missingCalls int
	deletedCalls int
	missingErr   error
}

func (h *recordingBlobDeleteHandler) OnBlobMissing() error {
	h.missingCalls++
	return h.missingErr
}

func (h *recordingBlobDeleteHandler) OnBlobDeleted() error {
	h.deletedCalls++
	return nil
}

var _ option.ResolvableDeleter = (*fakeBlobDeleter)(nil)
var _ metadata.BlobDeleteHandler = (*recordingBlobDeleteHandler)(nil)

func Test_deleteBlobWithDeleter_blobMissing(t *testing.T) {
	opts := &deleteBlobOptions{}
	opts.Reference = "sha256:missing"
	opts.RawReference = "example.com/repo@sha256:missing"
	opts.Force = true

	handler := &recordingBlobDeleteHandler{}
	deleter := &fakeBlobDeleter{resolveErr: errdef.ErrNotFound}

	err := deleteBlobWithDeleter(
		context.Background(),
		&cobra.Command{},
		opts,
		deleter,
		handler,
	)
	if err != nil {
		t.Fatalf("deleteBlobWithDeleter() error = %v, want nil", err)
	}
	if handler.missingCalls != 1 {
		t.Errorf("OnBlobMissing() calls = %d, want 1", handler.missingCalls)
	}
	if handler.deletedCalls != 0 {
		t.Errorf("OnBlobDeleted() calls = %d, want 0", handler.deletedCalls)
	}
}
