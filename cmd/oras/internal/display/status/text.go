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
	"sync"

	"oras.land/oras/internal/graph"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/output"
)

// TextPushHandler handles text status output for push events.
type TextPushHandler struct {
	printer   *output.Printer
	committed *sync.Map
	fetcher   content.Fetcher
}

// NewTextPushHandler returns a new handler for push command.
func NewTextPushHandler(printer *output.Printer, fetcher content.Fetcher) PushHandler {
	tch := TextPushHandler{
		printer:   printer,
		fetcher:   fetcher,
		committed: &sync.Map{},
	}
	return &tch
}

// OnFileLoading is called when a file is being prepared for upload.
func (ph *TextPushHandler) OnFileLoading(name string) error {
	return ph.printer.PrintVerbose("Preparing", name)
}

// OnEmptyArtifact is called when an empty artifact is being uploaded.
func (ph *TextPushHandler) OnEmptyArtifact() error {
	return ph.printer.Println("Uploading empty artifact")
}

// TrackTarget returns a tracked target.
func (ph *TextPushHandler) TrackTarget(gt oras.GraphTarget) (oras.GraphTarget, StopTrackTargetFunc, error) {
	return gt, discardStopTrack, nil
}

// OnCopySkipped is called when an object already exists.
func (ph *TextPushHandler) OnCopySkipped(_ context.Context, desc ocispec.Descriptor) error {
	ph.committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
	return ph.printer.PrintStatus(desc, PushPromptExists)
}

// PreCopy implements PreCopy of CopyHandler.
func (ph *TextPushHandler) PreCopy(_ context.Context, desc ocispec.Descriptor) error {
	return ph.printer.PrintStatus(desc, PushPromptUploading)
}

// PostCopy implements PostCopy of CopyHandler.
func (ph *TextPushHandler) PostCopy(ctx context.Context, desc ocispec.Descriptor) error {
	ph.committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
	successors, err := graph.FilteredSuccessors(ctx, desc, ph.fetcher, DeduplicatedFilter(ph.committed))
	if err != nil {
		return err
	}
	for _, successor := range successors {
		if err = ph.printer.PrintStatus(successor, PushPromptExists); err != nil {
			return err
		}
	}
	return ph.printer.PrintStatus(desc, PushPromptUploaded)
}

// NewTextAttachHandler returns a new handler for attach command.
func NewTextAttachHandler(printer *output.Printer, fetcher content.Fetcher) AttachHandler {
	return NewTextPushHandler(printer, fetcher)
}

// TextPullHandler handles text status output for pull events.
type TextPullHandler struct {
	printer *output.Printer
}

// TrackTarget implements PullHandler.
func (ph *TextPullHandler) TrackTarget(gt oras.GraphTarget) (oras.GraphTarget, StopTrackTargetFunc, error) {
	return gt, discardStopTrack, nil
}

// OnNodeDownloading implements PullHandler.
func (ph *TextPullHandler) OnNodeDownloading(desc ocispec.Descriptor) error {
	return ph.printer.PrintStatus(desc, PullPromptDownloading)
}

// OnNodeDownloaded implements PullHandler.
func (ph *TextPullHandler) OnNodeDownloaded(desc ocispec.Descriptor) error {
	return ph.printer.PrintStatus(desc, PullPromptDownloaded)
}

// OnNodeRestored implements PullHandler.
func (ph *TextPullHandler) OnNodeRestored(desc ocispec.Descriptor) error {
	return ph.printer.PrintStatus(desc, PullPromptRestored)
}

// OnNodeProcessing implements PullHandler.
func (ph *TextPullHandler) OnNodeProcessing(desc ocispec.Descriptor) error {
	return ph.printer.PrintStatus(desc, PullPromptProcessing)
}

// OnNodeSkipped implements PullHandler.
func (ph *TextPullHandler) OnNodeSkipped(desc ocispec.Descriptor) error {
	return ph.printer.PrintStatus(desc, PullPromptSkipped)
}

// NewTextPullHandler returns a new handler for pull command.
func NewTextPullHandler(printer *output.Printer) PullHandler {
	return &TextPullHandler{
		printer: printer,
	}
}

// TextCopyHandler handles text status output for push events.
type TextCopyHandler struct {
	printer   *output.Printer
	committed *sync.Map
	fetcher   content.Fetcher
}

// NewTextCopyHandler returns a new handler for push command.
func NewTextCopyHandler(printer *output.Printer, fetcher content.Fetcher) CopyHandler {
	return &TextCopyHandler{
		printer:   printer,
		fetcher:   fetcher,
		committed: &sync.Map{},
	}
}

// OnCopySkipped is called when an object already exists.
func (ch *TextCopyHandler) OnCopySkipped(_ context.Context, desc ocispec.Descriptor) error {
	ch.committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
	return ch.printer.PrintStatus(desc, copyPromptExists)
}

// PreCopy implements PreCopy of CopyHandler.
func (ch *TextCopyHandler) PreCopy(_ context.Context, desc ocispec.Descriptor) error {
	return ch.printer.PrintStatus(desc, copyPromptCopying)
}

// PostCopy implements PostCopy of CopyHandler.
func (ch *TextCopyHandler) PostCopy(ctx context.Context, desc ocispec.Descriptor) error {
	ch.committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
	deduplicated, err := graph.FilteredSuccessors(ctx, desc, ch.fetcher, DeduplicatedFilter(ch.committed))
	if err != nil {
		return err
	}
	for _, successor := range deduplicated {
		if err = ch.printer.PrintStatus(successor, copyPromptSkipped); err != nil {
			return err
		}
	}
	return ch.printer.PrintStatus(desc, copyPromptCopied)
}

// OnMounted implements OnMounted of CopyHandler.
func (ch *TextCopyHandler) OnMounted(_ context.Context, desc ocispec.Descriptor) error {
	ch.committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
	return ch.printer.PrintStatus(desc, copyPromptMounted)
}

// TextManifestIndexCreateHandler handles text status output for manifest index create events.
type TextManifestIndexCreateHandler struct {
	printer *output.Printer
}

// OnSourceManifestFetching implements ManifestIndexCreateHandler.
func (mich TextManifestIndexCreateHandler) OnSourceManifestFetching(source string) error {
	return mich.printer.Println(IndexPromptFetching, source)
}

// OnSourceManifestFetched implements ManifestIndexCreateHandler.
func (mich TextManifestIndexCreateHandler) OnSourceManifestFetched(source string) error {
	return mich.printer.Println(IndexPromptFetched, source)
}

// OnIndexPacked implements ManifestIndexCreateHandler.
func (mich TextManifestIndexCreateHandler) OnIndexPacked(shortDigest string) error {
	return mich.printer.Println(IndexPromptPacked, shortDigest, ocispec.MediaTypeImageIndex)
}

// OnIndexPushed implements ManifestIndexCreateHandler.
func (mich TextManifestIndexCreateHandler) OnIndexPushed(path string) error {
	return mich.printer.Println(IndexPromptPushed, path)
}

// NewTextManifestIndexCreateHandler returns a new handler for manifest index create command.
func NewTextManifestIndexCreateHandler(printer *output.Printer) ManifestIndexCreateHandler {
	tmich := TextManifestIndexCreateHandler{
		printer: printer,
	}
	return &tmich
}

// TextManifestIndexUpdateHandler handles text status output for manifest index update events.
type TextManifestIndexUpdateHandler struct {
	printer *output.Printer
}

// OnIndexFetching implements ManifestIndexUpdateHandler.
func (miuh TextManifestIndexUpdateHandler) OnIndexFetching(indexRef string) error {
	return miuh.printer.Println(IndexPromptFetching, indexRef)
}

// OnIndexFetched implements ManifestIndexUpdateHandler.
func (miuh TextManifestIndexUpdateHandler) OnIndexFetched(indexRef string, digest digest.Digest) error {
	if digest != "" {
		return miuh.printer.Println(IndexPromptFetched, digest, indexRef)
	}
	return miuh.printer.Println(IndexPromptFetched, indexRef)
}

// OnManifestFetching implements ManifestIndexUpdateHandler.
func (miuh TextManifestIndexUpdateHandler) OnManifestFetching(ref string) error {
	return miuh.printer.Println(IndexPromptFetching, ref)
}

// OnManifestFetched implements ManifestIndexUpdateHandler.
func (miuh TextManifestIndexUpdateHandler) OnManifestFetched(ref string, digest digest.Digest) error {
	if digest != "" {
		return miuh.printer.Println(IndexPromptFetched, digest, ref)
	}
	return miuh.printer.Println(IndexPromptFetched, ref)
}

// OnManifestRemoved implements ManifestIndexUpdateHandler.
func (miuh TextManifestIndexUpdateHandler) OnManifestRemoved(digest digest.Digest) error {
	return miuh.printer.Println(IndexPromptRemoved, digest)
}

// OnManifestAdded implements ManifestIndexUpdateHandler.
func (miuh TextManifestIndexUpdateHandler) OnManifestAdded(ref string, digest digest.Digest) error {
	if digest != "" {
		return miuh.printer.Println(IndexPromptAdded, digest, ref)
	}
	return miuh.printer.Println(IndexPromptAdded, ref)
}

// OnIndexMerged implements ManifestIndexUpdateHandler.
func (miuh TextManifestIndexUpdateHandler) OnIndexMerged(ref string, digest digest.Digest) error {
	if digest != "" {
		return miuh.printer.Println(IndexPromptMerged, digest, ref)
	}
	return miuh.printer.Println(IndexPromptMerged, ref)
}

// OnIndexUpdated implements ManifestIndexUpdateHandler.
func (miuh TextManifestIndexUpdateHandler) OnIndexUpdated(source string) error {
	return miuh.printer.Println(IndexPromptUpdated, source)
}

// OnIndexPushed implements ManifestIndexUpdateHandler.
func (miuh TextManifestIndexUpdateHandler) OnIndexPushed(source string) error {
	return miuh.printer.Println(IndexPromptPushed, source)
}

// NewTextManifestIndexUpdateHandler returns a new handler for manifest index create command.
func NewTextManifestIndexUpdateHandler(printer *output.Printer) ManifestIndexUpdateHandler {
	miuh := TextManifestIndexUpdateHandler{
		printer: printer,
	}
	return &miuh
}
