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
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/status"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
)

func Test_runRestore(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	given := &restoreOptions{}
	got := runRestore(cmd, given)
	want := "failed to parse source target : found empty file path"
	if got == nil || !strings.Contains(got.Error(), want) {
		t.Fatalf("got %v, want %s", got, want)
	}

	given = &restoreOptions{
		input: t.TempDir(),
	}
	got = runRestore(cmd, given)
	if got != nil {
		t.Fatalf("got %v, want nil", got)
	}
}

func Test_recursiveRestore(t *testing.T) {
	ctx := context.Background()

	sourceDirectory := "./testdata"
	src, err := oci.New(sourceDirectory)
	if err != nil {
		t.Fatalf("failed to new oci source %s: %v", sourceDirectory, err)
	}

	dst, err := oci.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to new oci source: %v", err)
	}

	sourceDirectory, err = filepath.Abs(sourceDirectory)
	if err != nil {
		t.Fatalf("failed to absolute source directory %s: %v", sourceDirectory, err)
	}

	root, err := oras.Resolve(ctx, src, "localhost:15000/artifact/two:v1", oras.DefaultResolveOptions)
	if err != nil {
		t.Fatalf("failed to resolve source %s: %v", sourceDirectory, err)
	}

	mock := mockHandler{}
	given := oras.DefaultExtendedCopyOptions
	given.OnCopySkipped = mock.OnCopySkipped
	given.OnMounted = mock.OnMounted
	given.PostCopy = mock.PostCopy
	given.PreCopy = mock.PreCopy
	got := recursiveRestore(ctx, src, dst, root, given)
	if got != nil {
		t.Fatalf("got %v, want nil", got)
	}

}

type mockHandler struct {
	lock                    sync.Mutex
	onCopiedSource          string
	onCopiedDestination     string
	onCopiedError           error
	onCopySkippedDescriptor ocispec.Descriptor
	onCopySkippedError      error
	onMountedDescriptor     ocispec.Descriptor
	onMountedError          error
	onTaggedDescriptor      ocispec.Descriptor
	onTaggedTag             string
	postCopyDescriptor      ocispec.Descriptor
	postCopyError           error
	onTaggedError           error
	preCopyDescriptor       ocispec.Descriptor
	preCopyError            error
	startTrackingError      error
	stopTrackingError       error
}

var _ metadata.BackupHandler = &mockHandler{}
var _ status.BackupHandler = &mockHandler{}

func (mock *mockHandler) OnCopied(source, destination string) error {
	mock.onCopiedSource = source
	mock.onCopiedDestination = destination
	return mock.onCopiedError
}

func (mock *mockHandler) OnCopySkipped(_ context.Context, desc ocispec.Descriptor) error {
	mock.onCopySkippedDescriptor = desc
	return mock.onCopySkippedError
}

func (mock *mockHandler) OnMounted(_ context.Context, desc ocispec.Descriptor) error {
	mock.onMountedDescriptor = desc
	return mock.onMountedError
}

func (mock *mockHandler) OnTagged(desc ocispec.Descriptor, tag string) error {
	mock.onTaggedDescriptor = desc
	mock.onTaggedTag = tag
	return mock.onTaggedError
}

func (mock *mockHandler) PostCopy(_ context.Context, desc ocispec.Descriptor) error {
	mock.lock.Lock()
	defer mock.lock.Unlock()
	mock.postCopyDescriptor = desc
	return mock.postCopyError
}

func (mock *mockHandler) PreCopy(_ context.Context, desc ocispec.Descriptor) error {
	mock.lock.Lock()
	defer mock.lock.Unlock()
	mock.preCopyDescriptor = desc
	return mock.preCopyError
}

func (mock *mockHandler) StartTracking(gt oras.GraphTarget) (oras.GraphTarget, error) {
	return gt, mock.startTrackingError
}

func (mock *mockHandler) StopTracking() error {
	return mock.stopTrackingError
}
