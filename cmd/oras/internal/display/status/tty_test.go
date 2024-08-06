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

package status

import (
	"context"
	"io"
	"os"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
)

func TestTTYPushHandler_OnFileLoading(t *testing.T) {
	ph := NewTTYPushHandler(os.Stdout)
	if ph.OnFileLoading("test") != nil {
		t.Error("OnFileLoading() should not return an error")
	}
}

func TestTTYPushHandler_OnEmptyArtifact(t *testing.T) {
	ph := NewTTYAttachHandler(os.Stdout)
	if ph.OnEmptyArtifact() != nil {
		t.Error("OnEmptyArtifact() should not return an error")
	}
}

func TestTTYPushHandler_TrackTarget_invalidTTY(t *testing.T) {
	ph := NewTTYPushHandler(os.Stdin)
	if _, _, err := ph.TrackTarget(nil); err == nil {
		t.Error("TrackTarget() should return an error for non-tty file")
	}
}

func TestTTYPullHandler_OnNodeDownloading(t *testing.T) {
	ph := NewTTYPullHandler(nil)
	if err := ph.OnNodeDownloading(ocispec.Descriptor{}); err != nil {
		t.Error("OnNodeDownloading() should not return an error")
	}
}

func TestTTYPullHandler_OnNodeDownloaded(t *testing.T) {
	ph := NewTTYPullHandler(nil)
	if err := ph.OnNodeDownloaded(ocispec.Descriptor{}); err != nil {
		t.Error("OnNodeDownloaded() should not return an error")
	}
}

func TestTTYPullHandler_OnNodeProcessing(t *testing.T) {
	ph := NewTTYPullHandler(nil)
	if err := ph.OnNodeProcessing(ocispec.Descriptor{}); err != nil {
		t.Error("OnNodeProcessing() should not return an error")
	}
}

// ErrorFetcher implements content.Fetcher.
type ErrorFetcher struct{}

// Fetch returns an error.
func (f *ErrorFetcher) Fetch(context.Context, ocispec.Descriptor) (io.ReadCloser, error) {
	return nil, wantedError
}

func TestTTYPushHandler_errGetSuccessor(t *testing.T) {
	ph := NewTTYPushHandler(nil)
	opts := oras.CopyGraphOptions{}
	ph.UpdateCopyOptions(&opts, &ErrorFetcher{})
	if err := opts.PostCopy(context.Background(), ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
	}); err != wantedError {
		t.Error("PostCopy() should return expected error")
	}
}

// ErrorPromptor mocks trackable GraphTarget.
type ErrorPromptor struct {
	oras.GraphTarget
	io.Closer
}

// Prompt mocks an errored prompt.
func (p *ErrorPromptor) Prompt(ocispec.Descriptor, string) error {
	return wantedError
}

func TestTTYPushHandler_errPrompt(t *testing.T) {
	ph := TTYPushHandler{
		tracked: &ErrorPromptor{},
	}
	opts := oras.CopyGraphOptions{}
	ph.UpdateCopyOptions(&opts, memStore)
	if err := opts.OnCopySkipped(ctx, layerDesc); err != wantedError {
		t.Error("OnCopySkipped() should return expected error")
	}
	// test
	if err := opts.PostCopy(context.Background(), manifestDesc); err != wantedError {
		t.Error("PostCopy() should return expected error")
	}
}
