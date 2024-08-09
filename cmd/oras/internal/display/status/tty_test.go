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
	"errors"
	"os"
	"sync"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/internal/testutils"
)

func TestTTYPushHandler_OnFileLoading(t *testing.T) {
	ph := NewTTYPushHandler(os.Stdout, mockFetcher.Fetcher)
	if ph.OnFileLoading("test") != nil {
		t.Error("OnFileLoading() should not return an error")
	}
}

func TestTTYPushHandler_OnEmptyArtifact(t *testing.T) {
	ph := NewTTYAttachHandler(os.Stdout, mockFetcher.Fetcher)
	if ph.OnEmptyArtifact() != nil {
		t.Error("OnEmptyArtifact() should not return an error")
	}
}

func TestTTYPushHandler_TrackTarget_invalidTTY(t *testing.T) {
	ph := NewTTYPushHandler(os.Stdin, mockFetcher.Fetcher)
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

func TestTTYPushHandler_PostCopy(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	committed := &sync.Map{}
	committed.Store(fetcher.ImageLayer.Digest.String(), fetcher.ImageLayer.Annotations[ocispec.AnnotationTitle])
	ph := &TTYPushHandler{
		tracked:   &testutils.PromptDiscarder{},
		committed: committed,
		fetcher:   fetcher.Fetcher,
	}
	if err := ph.PostCopy(ctx, fetcher.OciImage); err != nil {
		t.Errorf("unexpected error from PostCopy(): %v", err)
	}
}

func TestTTYPushHandler_PostCopy_errGetSuccessor(t *testing.T) {
	errorFetcher := testutils.NewErrorFetcher()
	ph := NewTTYPushHandler(nil, errorFetcher)
	err := ph.PostCopy(ctx, ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
	})
	if err.Error() != errorFetcher.ExpectedError.Error() {
		t.Errorf("PostCopy() should return expected error got %v", err.Error())
	}
}

func TestTTYPushHandler_PostCopy_errPrompt(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	committed := &sync.Map{}
	committed.Store(fetcher.ImageLayer.Digest.String(), fetcher.ImageLayer.Annotations[ocispec.AnnotationTitle]+"1")
	wantedError := errors.New("wanted error")
	ph := &TTYPushHandler{
		tracked:   testutils.NewErrorPrompt(wantedError),
		committed: committed,
		fetcher:   fetcher.Fetcher,
	}
	if err := ph.PostCopy(ctx, fetcher.OciImage); err != wantedError {
		t.Errorf("PostCopy() should return expected error got %v", err)
	}
}

func TestTTYPushHandler_OnCopySkipped(t *testing.T) {
	ph := &TTYPushHandler{
		tracked:   &testutils.PromptDiscarder{},
		committed: &sync.Map{},
	}
	if err := ph.OnCopySkipped(ctx, ocispec.Descriptor{}); err != nil {
		t.Error("OnCopySkipped() should not return an error")
	}
}
