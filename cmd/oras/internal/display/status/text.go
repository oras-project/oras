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
	"fmt"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/display/utils"
)

// TextPushHandler handles text status output for push events.
type TextPushHandler struct {
	verbose bool
}

// NewTextPushHandler returns a new handler for push command.
func NewTextPushHandler(verbose bool) PushHandler {
	return &TextPushHandler{
		verbose: verbose,
	}
}

// OnFileLoading is called when a file is being prepared for upload.
func (ph *TextPushHandler) OnFileLoading(name string) error {
	if !ph.verbose {
		return nil
	}
	_, err := fmt.Println("Preparing", name)
	return err
}

// OnEmptyArtifact is called when an empty artifact is being uploaded.
func (ph *TextPushHandler) OnEmptyArtifact() error {
	_, err := fmt.Println("Uploading empty artifact")
	return err
}

// TrackTarget returns a tracked target.
func (ph *TextPushHandler) TrackTarget(gt oras.GraphTarget) (oras.GraphTarget, error) {
	return gt, nil
}

// UpdatePushCopyOptions adds status update to the copy options.
func (ph *TextPushHandler) UpdatePushCopyOptions(opts *oras.CopyGraphOptions, fetcher content.Fetcher) {
	const (
		promptSkipped   = "Skipped  "
		promptUploaded  = "Uploaded "
		promptExists    = "Exists   "
		promptUploading = "Uploading"
	)
	committed := &sync.Map{}
	opts.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		return PrintStatus(desc, promptExists, ph.verbose)
	}
	opts.PreCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		return PrintStatus(desc, promptUploading, ph.verbose)
	}
	opts.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		if err := PrintSuccessorStatus(ctx, desc, fetcher, committed, StatusPrinter(promptSkipped, ph.verbose)); err != nil {
			return err
		}
		return PrintStatus(desc, promptUploaded, ph.verbose)
	}
}

// NewTextAttachHandler returns a new handler for attach command.
func NewTextAttachHandler(verbose bool) AttachHandler {
	return NewTextPushHandler(verbose)
}

// TextPullHandler handles text status output for pull events.
type TextPullHandler struct {
	fetcher content.Fetcher
	verbose bool
}

func (ph *TextPullHandler) printOnce(printed *sync.Map, s ocispec.Descriptor, msg string) error {
	if _, loaded := printed.LoadOrStore(utils.GenerateContentKey(s), true); loaded {
		return nil
	}
	return PrintStatus(s, msg, ph.verbose)
}

func (ph *TextPullHandler) UpdatePullCopyOptions(opts *oras.CopyGraphOptions, printed *sync.Map, includeSubject bool, configPath string, configMediaType string) {
	const (
		promptDownloading = "Downloading"
		promptSkipped     = "Skipped    "
		promptRestored    = "Restored   "
		promptDownloaded  = "Downloaded "
	)

	opts.PreCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		if _, ok := printed.LoadOrStore(utils.GenerateContentKey(desc), true); ok {
			return nil
		}
		return PrintStatus(desc, promptDownloading, ph.verbose)
	}
	opts.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		// restore named but deduplicated successor nodes
		successors, err := content.Successors(ctx, ph.fetcher, desc)
		if err != nil {
			return err
		}
		for _, s := range successors {
			if _, ok := s.Annotations[ocispec.AnnotationTitle]; ok {
				if err := ph.printOnce(printed, s, promptRestored); err != nil {
					return err
				}
			}
		}
		name, ok := desc.Annotations[ocispec.AnnotationTitle]
		if !ok {
			if !ph.verbose {
				return nil
			}
			name = desc.MediaType
		}
		printed.Store(utils.GenerateContentKey(desc), true)
		return Print(promptDownloaded, ShortDigest(desc), name)
	}
}

// TrackTarget returns a tracked target.
func (ph *TextPullHandler) TrackTarget(gt oras.GraphTarget) (oras.GraphTarget, error) {
	ph.fetcher = gt
	return gt, nil
}

// StopTracking stop tracked target.
func (ph *TextPullHandler) StopTracking() {}

// OnNodeDownloading implements PullHandler.
func (ph *TextPullHandler) OnNodeDownloading(desc ocispec.Descriptor) error {
	return PrintStatus(desc, "Downloading", ph.verbose)
}

// OnNodeProcessing implements PullHandler.
func (ph *TextPullHandler) OnNodeProcessing(desc ocispec.Descriptor) error {
	return PrintStatus(desc, "Processing ", ph.verbose)
}

// OnNodeProcessing implements PullHandler.
func (ph *TextPullHandler) OnNodeSkipped(printed *sync.Map, desc ocispec.Descriptor) error {
	return ph.printOnce(printed, desc, "Skipped    ")
}

// NewTextPullHandler returns a new handler for pull command.
func NewTextPullHandler(verbose bool) PullHandler {
	return &TextPullHandler{
		verbose: verbose,
	}
}
