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
	"os"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/display/status/track"
	"oras.land/oras/cmd/oras/internal/display/utils"
)

// TTYPushHandler handles TTY status output for push command.
type TTYPushHandler struct {
	tty     *os.File
	tracked track.GraphTarget
}

// NewTTYPushHandler returns a new handler for push status events.
func NewTTYPushHandler(tty *os.File) PushHandler {
	return &TTYPushHandler{
		tty: tty,
	}
}

// OnFileLoading is called before loading a file.
func (ph *TTYPushHandler) OnFileLoading(name string) error {
	return nil
}

// OnEmptyArtifact is called when no file is loaded for an artifact push.
func (ph *TTYPushHandler) OnEmptyArtifact() error {
	return nil
}

// TrackTarget returns a tracked target.
func (ph *TTYPushHandler) TrackTarget(gt oras.GraphTarget) (oras.GraphTarget, StopTrackTargetFunc, error) {
	const (
		promptUploaded  = "Uploaded "
		promptUploading = "Uploading"
	)
	tracked, err := track.NewTarget(gt, promptUploading, promptUploaded, ph.tty)
	if err != nil {
		return nil, nil, err
	}
	ph.tracked = tracked
	return tracked, tracked.Close, nil
}

// UpdateCopyOptions adds TTY status output to the copy options.
func (ph *TTYPushHandler) UpdateCopyOptions(opts *oras.CopyGraphOptions, fetcher content.Fetcher) {
	const (
		promptSkipped = "Skipped  "
		promptExists  = "Exists   "
	)
	committed := &sync.Map{}
	opts.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		return ph.tracked.Prompt(desc, promptExists)
	}
	opts.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		return utils.PrintSuccessorStatus(ctx, desc, fetcher, committed, func(d ocispec.Descriptor) error {
			return ph.tracked.Prompt(d, promptSkipped)
		})
	}
}

// NewTTYAttachHandler returns a new handler for attach status events.
func NewTTYAttachHandler(tty *os.File) AttachHandler {
	return NewTTYPushHandler(tty)
}

// TTYPullHandler handles TTY status output for pull events.
type TTYPullHandler struct {
	tty     *os.File
	tracked track.GraphTarget
}

// NewTTYPullHandler returns a new handler for Pull status events.
func NewTTYPullHandler(tty *os.File) PullHandler {
	return &TTYPullHandler{
		tty: tty,
	}
}

// OnNodeDownloading implements PullHandler.
func (ph *TTYPullHandler) OnNodeDownloading(desc ocispec.Descriptor) error {
	return nil
}

// OnNodeDownloaded implements PullHandler.
func (ph *TTYPullHandler) OnNodeDownloaded(desc ocispec.Descriptor) error {
	return nil
}

// OnNodeProcessing implements PullHandler.
func (ph *TTYPullHandler) OnNodeProcessing(desc ocispec.Descriptor) error {
	return nil
}

// OnNodeRestored implements PullHandler.
func (ph *TTYPullHandler) OnNodeRestored(desc ocispec.Descriptor) error {
	return ph.tracked.Prompt(desc, PullPromptRestored)
}

// OnNodeProcessing implements PullHandler.
func (ph *TTYPullHandler) OnNodeSkipped(desc ocispec.Descriptor) error {
	return ph.tracked.Prompt(desc, PullPromptSkipped)
}

// TrackTarget returns a tracked target.
func (ph *TTYPullHandler) TrackTarget(gt oras.GraphTarget) (oras.GraphTarget, StopTrackTargetFunc, error) {
	tracked, err := track.NewTarget(gt, PullPromptDownloading, PullPromptPulled, ph.tty)
	if err != nil {
		return nil, nil, err
	}
	ph.tracked = tracked
	return tracked, tracked.Close, nil
}
