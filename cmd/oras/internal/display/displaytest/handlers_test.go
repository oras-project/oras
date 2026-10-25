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

package displaytest

import (
	"context"
	"errors"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras/cmd/oras/internal/option"
)

func TestRecordingHandlers(t *testing.T) {
	recorder := &CallRecorder{}
	ctx := context.Background()
	desc := ocispec.Descriptor{}
	target := memory.New()
	stopErr := errors.New("stop tracking")

	attachStatus := &RecordingAttachStatusHandler{Recorder: recorder, TrackErr: stopErr}
	if _, stop, err := attachStatus.TrackTarget(target); !errors.Is(err, stopErr) {
		t.Fatalf("TrackTarget() error = %v, want %v", err, stopErr)
	} else if err := stop(); err != nil {
		t.Fatal(err)
	}
	if err := attachStatus.OnFileLoading(""); err != nil {
		t.Fatal(err)
	}
	if err := attachStatus.OnCopySkipped(ctx, desc); err != nil {
		t.Fatal(err)
	}
	if err := attachStatus.PreCopy(ctx, desc); err != nil {
		t.Fatal(err)
	}
	if err := attachStatus.PostCopy(ctx, desc); err != nil {
		t.Fatal(err)
	}
	attachMetadata := &RecordingAttachMetadataHandler{Recorder: recorder, RenderErr: stopErr}
	attachMetadata.OnAttached(&option.Target{}, desc, desc)
	if err := attachMetadata.Render(); !errors.Is(err, stopErr) {
		t.Fatalf("Render() error = %v, want %v", err, stopErr)
	}

	copyStatus := &RecordingCopyStatusHandler{Recorder: recorder}
	for _, call := range []func() error{
		func() error { return copyStatus.OnCopySkipped(ctx, desc) },
		func() error { return copyStatus.PreCopy(ctx, desc) },
		func() error { return copyStatus.PostCopy(ctx, desc) },
		func() error { return copyStatus.OnMounted(ctx, desc) },
	} {
		if err := call(); err != nil {
			t.Fatal(err)
		}
	}
	copyMetadata := &RecordingCopyMetadataHandler{Recorder: recorder, CopiedErr: stopErr, RenderErr: stopErr}
	if err := copyMetadata.OnCopied(&option.BinaryTarget{}, desc); !errors.Is(err, stopErr) {
		t.Fatalf("OnCopied() error = %v, want %v", err, stopErr)
	}
	if err := copyMetadata.Render(); !errors.Is(err, stopErr) {
		t.Fatalf("Render() error = %v, want %v", err, stopErr)
	}

	fetchContent := &RecordingManifestFetchContentHandler{Recorder: recorder, ContentFetchedErr: stopErr}
	if err := fetchContent.OnContentFetched(desc, nil); !errors.Is(err, stopErr) {
		t.Fatalf("OnContentFetched() error = %v, want %v", err, stopErr)
	}
	fetchMetadata := &RecordingManifestFetchMetadataHandler{Recorder: recorder, FetchedErr: stopErr}
	if err := fetchMetadata.OnFetched("ref", desc, nil); !errors.Is(err, stopErr) {
		t.Fatalf("OnFetched() error = %v, want %v", err, stopErr)
	}

	pullStatus := &RecordingPullStatusHandler{Recorder: recorder, TrackErr: stopErr, NodeProcessingErr: stopErr, NodeDownloadingErr: stopErr}
	if _, stop, err := pullStatus.TrackTarget(target); !errors.Is(err, stopErr) {
		t.Fatalf("TrackTarget() error = %v, want %v", err, stopErr)
	} else if err := stop(); err != nil {
		t.Fatal(err)
	}
	if err := pullStatus.OnNodeProcessing(desc); !errors.Is(err, stopErr) {
		t.Fatalf("OnNodeProcessing() error = %v, want %v", err, stopErr)
	}
	if err := pullStatus.OnNodeDownloading(desc); !errors.Is(err, stopErr) {
		t.Fatalf("OnNodeDownloading() error = %v, want %v", err, stopErr)
	}
	for _, call := range []func() error{
		func() error { return pullStatus.OnNodeDownloaded(desc) },
		func() error { return pullStatus.OnNodeRestored(desc) },
		func() error { return pullStatus.OnNodeSkipped(desc) },
	} {
		if err := call(); err != nil {
			t.Fatal(err)
		}
	}

	pushStatus := &RecordingPushStatusHandler{Recorder: recorder, FileLoadingErr: stopErr}
	if err := pushStatus.OnFileLoading("file"); !errors.Is(err, stopErr) {
		t.Fatalf("OnFileLoading() error = %v, want %v", err, stopErr)
	}
	pushMetadata := &RecordingPushMetadataHandler{Recorder: recorder, CopiedErr: stopErr, TaggedErr: stopErr, RenderErr: stopErr}
	if err := pushMetadata.OnCopied(&option.Target{}, desc); !errors.Is(err, stopErr) {
		t.Fatalf("OnCopied() error = %v, want %v", err, stopErr)
	}
	if err := pushMetadata.OnTagged(desc, "tag"); !errors.Is(err, stopErr) {
		t.Fatalf("OnTagged() error = %v, want %v", err, stopErr)
	}
	if err := pushMetadata.Render(); !errors.Is(err, stopErr) {
		t.Fatalf("Render() error = %v, want %v", err, stopErr)
	}

	restoreStatus := &RecordingRestoreStatusHandler{Recorder: recorder, PreCopyErr: stopErr}
	if _, err := restoreStatus.StartTracking(target); err != nil {
		t.Fatal(err)
	}
	if err := restoreStatus.StopTracking(); err != nil {
		t.Fatal(err)
	}
	for _, call := range []func() error{
		func() error { return restoreStatus.OnCopySkipped(ctx, desc) },
		func() error { return restoreStatus.PreCopy(ctx, desc) },
		func() error { return restoreStatus.PostCopy(ctx, desc) },
	} {
		_ = call()
	}
	restoreMetadata := &RecordingRestoreMetadataHandler{Recorder: recorder, TagsFoundErr: stopErr, ArtifactPushedErr: stopErr, CompletedErr: stopErr}
	if err := restoreMetadata.OnTarLoaded("tar", 1); err != nil {
		t.Fatal(err)
	}
	if err := restoreMetadata.OnTagsFound(nil); !errors.Is(err, stopErr) {
		t.Fatalf("OnTagsFound() error = %v, want %v", err, stopErr)
	}
	if err := restoreMetadata.OnArtifactPushed("tag", 1); !errors.Is(err, stopErr) {
		t.Fatalf("OnArtifactPushed() error = %v, want %v", err, stopErr)
	}
	if err := restoreMetadata.OnRestoreCompleted(1, "repo", 0); !errors.Is(err, stopErr) {
		t.Fatalf("OnRestoreCompleted() error = %v, want %v", err, stopErr)
	}
	if len(recorder.Snapshot()) == 0 {
		t.Fatal("handlers recorded no calls")
	}
}
