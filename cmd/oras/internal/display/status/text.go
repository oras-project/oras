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

// UpdateCopyOptions adds status update to the copy options.
func (ph *TextPushHandler) UpdateCopyOptions(opts *oras.CopyGraphOptions, fetcher content.Fetcher) {
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

// TextTagHandler handles text metadata output for tag events.
type TextTagHandler struct{}

// OnTagged implements metadata.TextTagHandler.
func (t TextTagHandler) OnTagged(tag string) error {
	return Print("Tagged", tag)
}

// PreTagging implements metadata.TextTagHandler.
func (t TextTagHandler) PreTagging(reference string) {
	_ = Print("Tagging", reference)
}

// NewTextTagHandler returns a new handler for tag events.
func NewTextTagHandler() TagHandler {
	return TextTagHandler{}
}
