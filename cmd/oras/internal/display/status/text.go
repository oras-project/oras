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
	"oras.land/oras/internal/graph"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/output"
)

// TextPushHandler handles text status output for push events.
type TextPushHandler struct {
	printer *output.Printer
}

// NewTextPushHandler returns a new handler for push command.
func NewTextPushHandler(printer *output.Printer) PushHandler {
	return &TextPushHandler{
		printer: printer,
	}
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

// UpdateCopyOptions adds status update to the copy options.
func (ph *TextPushHandler) UpdateCopyOptions(opts *oras.CopyGraphOptions, fetcher content.Fetcher) {
	committed := &sync.Map{}
	opts.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		return ph.printer.PrintStatus(desc, PushPromptExists)
	}
	opts.PreCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		return ph.printer.PrintStatus(desc, PushPromptUploading)
	}
	opts.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		successors, err := graph.FilteredSuccessors(ctx, desc, fetcher, DeduplicatedFilter(committed))
		if err != nil {
			return err
		}
		for _, successor := range successors {
			if err = ph.printer.PrintStatus(successor, PushPromptSkipped); err != nil {
				return err
			}
		}
		return ph.printer.PrintStatus(desc, PushPromptUploaded)
	}
}

// NewTextAttachHandler returns a new handler for attach command.
func NewTextAttachHandler(printer *output.Printer) AttachHandler {
	return NewTextPushHandler(printer)
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
	successors, err := graph.FilteredSuccessors(ctx, desc, ch.fetcher, DeduplicatedFilter(ch.committed))
	if err != nil {
		return err
	}
	for _, successor := range successors {
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
