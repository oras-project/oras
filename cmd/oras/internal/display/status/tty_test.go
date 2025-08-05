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
	"oras.land/oras-go/v2/content/memory"
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

func TestNewTTYBackupHandler(t *testing.T) {
	handler := NewTTYBackupHandler(os.Stdout, nil)
	if handler == nil {
		t.Error("NewTTYBackupHandler() should not return nil")
	}
}

func TestTTYBackupHandler_StartTracking_invalidTTY(t *testing.T) {
	bh := NewTTYBackupHandler(os.Stdin, nil)
	gt := memory.New()
	if _, err := bh.StartTracking(gt); err == nil {
		t.Error("StartTracking() should return an error for non-tty file")
	}
}

func TestTTYBackupHandler_OnCopySkipped(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	bh := &TTYBackupHandler{
		tracked:   &testutils.PromptDiscarder{}, // Keep PromptDiscarder here for Report method
		committed: &sync.Map{},
		fetcher:   fetcher.Fetcher,
	}
	if err := bh.OnCopySkipped(ctx, fetcher.ImageLayer); err != nil {
		t.Errorf("OnCopySkipped() should not return an error: %v", err)
	}

	// Verify that the descriptor is stored in the committed map
	if _, ok := bh.committed.Load(fetcher.ImageLayer.Digest.String()); !ok {
		t.Error("OnCopySkipped() should store the descriptor in the committed map")
	}
}

func TestTTYBackupHandler_PreCopy(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	bh := &TTYBackupHandler{}
	if err := bh.PreCopy(ctx, fetcher.ImageLayer); err != nil {
		t.Errorf("PreCopy() should not return an error: %v", err)
	}
}

func TestTTYBackupHandler_PostCopy(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	committed := &sync.Map{}
	committed.Store(fetcher.ImageLayer.Digest.String(), fetcher.ImageLayer.Annotations[ocispec.AnnotationTitle])
	bh := &TTYBackupHandler{
		tracked:   &testutils.PromptDiscarder{},
		committed: committed,
		fetcher:   fetcher.Fetcher,
	}
	if err := bh.PostCopy(ctx, fetcher.OciImage); err != nil {
		t.Errorf("unexpected error from PostCopy(): %v", err)
	}
}

func TestTTYBackupHandler_PostCopy_errGetSuccessor(t *testing.T) {
	errorFetcher := testutils.NewErrorFetcher()
	prompt := &testutils.PromptDiscarder{}
	bh := &TTYBackupHandler{
		tracked:   prompt,
		committed: &sync.Map{},
		fetcher:   errorFetcher,
	}

	err := bh.PostCopy(ctx, ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
	})

	if err == nil || err.Error() != errorFetcher.ExpectedError.Error() {
		t.Errorf("PostCopy() should return expected error got %v", err.Error())
	}
}

func TestTTYBackupHandler_PostCopy_errPrompt(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	committed := &sync.Map{}
	committed.Store(fetcher.ImageLayer.Digest.String(), fetcher.ImageLayer.Annotations[ocispec.AnnotationTitle]+"1")
	wantedError := errors.New("wanted error")
	bh := &TTYBackupHandler{
		tracked:   testutils.NewErrorPrompt(wantedError),
		committed: committed,
		fetcher:   fetcher.Fetcher,
	}
	if err := bh.PostCopy(ctx, fetcher.OciImage); err != wantedError {
		t.Errorf("PostCopy() should return expected error got %v", err)
	}
}

func TestNewTTYRestoreHandler(t *testing.T) {
	handler := NewTTYRestoreHandler(os.Stdout, nil)
	if handler == nil {
		t.Error("NewTTYRestoreHandler() should not return nil")
	}
}

func TestTTYRestoreHandler_StartTracking_invalidTTY(t *testing.T) {
	rh := NewTTYRestoreHandler(os.Stdin, nil)
	gt := memory.New()
	if _, err := rh.StartTracking(gt); err == nil {
		t.Error("StartTracking() should return an error for non-tty file")
	}
}

func TestTTYRestoreHandler_OnCopySkipped(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	rh := &TTYRestoreHandler{
		tracked:   &testutils.PromptDiscarder{},
		committed: &sync.Map{},
		fetcher:   fetcher.Fetcher,
	}
	if err := rh.OnCopySkipped(ctx, fetcher.ImageLayer); err != nil {
		t.Errorf("OnCopySkipped() should not return an error: %v", err)
	}

	// Verify that the descriptor is stored in the committed map
	if _, ok := rh.committed.Load(fetcher.ImageLayer.Digest.String()); !ok {
		t.Error("OnCopySkipped() should store the descriptor in the committed map")
	}
}

func TestTTYRestoreHandler_PreCopy(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	rh := &TTYRestoreHandler{}
	if err := rh.PreCopy(ctx, fetcher.ImageLayer); err != nil {
		t.Errorf("PreCopy() should not return an error: %v", err)
	}
}

func TestTTYRestoreHandler_PostCopy(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	committed := &sync.Map{}
	committed.Store(fetcher.ImageLayer.Digest.String(), fetcher.ImageLayer.Annotations[ocispec.AnnotationTitle])
	rh := &TTYRestoreHandler{
		tracked:   &testutils.PromptDiscarder{},
		committed: committed,
		fetcher:   fetcher.Fetcher,
	}
	if err := rh.PostCopy(ctx, fetcher.OciImage); err != nil {
		t.Errorf("unexpected error from PostCopy(): %v", err)
	}
}

func TestTTYRestoreHandler_PostCopy_errGetSuccessor(t *testing.T) {
	errorFetcher := testutils.NewErrorFetcher()
	prompt := &testutils.PromptDiscarder{}
	rh := &TTYRestoreHandler{
		tracked:   prompt,
		committed: &sync.Map{},
		fetcher:   errorFetcher,
	}

	err := rh.PostCopy(ctx, ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
	})

	if err == nil || err.Error() != errorFetcher.ExpectedError.Error() {
		t.Errorf("PostCopy() should return expected error got %v", err)
	}
}

func TestTTYRestoreHandler_PostCopy_errPrompt(t *testing.T) {
	fetcher := testutils.NewMockFetcher()
	committed := &sync.Map{}
	committed.Store(fetcher.ImageLayer.Digest.String(), fetcher.ImageLayer.Annotations[ocispec.AnnotationTitle]+"1")
	wantedError := errors.New("wanted error")
	rh := &TTYRestoreHandler{
		tracked:   testutils.NewErrorPrompt(wantedError),
		committed: committed,
		fetcher:   fetcher.Fetcher,
	}
	if err := rh.PostCopy(ctx, fetcher.OciImage); err != wantedError {
		t.Errorf("PostCopy() should return expected error got %v", err)
	}
}
