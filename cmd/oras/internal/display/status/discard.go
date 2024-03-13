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
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
)

// DiscardHandler is a no-op handler that discards all status updates.
type DiscardHandler struct{}

// NewDiscardHandler returns a new no-op handler.
func NewDiscardHandler() DiscardHandler {
	return DiscardHandler{}
}

// OnFileLoading is called before a file is being loaded.
func (DiscardHandler) OnFileLoading(name string) error {
	return nil
}

// OnEmptyArtifact is called when no file is loaded for an artifact push.
func (DiscardHandler) OnEmptyArtifact() error {
	return nil
}

// TrackTarget returns a target with status tracking
func (DiscardHandler) TrackTarget(gt oras.GraphTarget) (oras.GraphTarget, error) {
	return gt, nil
}

// StopTracking stops tracking the status of the target.
func (DiscardHandler) StopTracking() {}

// UpdatePushCopyOptions updates the copy options for the artifact push.
func (DiscardHandler) UpdatePushCopyOptions(opts *oras.CopyGraphOptions, fetcher content.Fetcher) {}

// UpdatePushCopyOptions updates the copy options for the artifact pull.
func (DiscardHandler) UpdatePullCopyOptions(opts *oras.CopyGraphOptions, printed *sync.Map, includeSubject bool, configPath string, configMediaType string) {
}

// OnNodeDownloading implements PullHandler.
func (DiscardHandler) OnNodeDownloading(desc ocispec.Descriptor) error {
	return nil
}

// OnNodeProcessing implements PullHandler.
func (DiscardHandler) OnNodeProcessing(desc ocispec.Descriptor) error {
	return nil
}

// OnNodeProcessing implements PullHandler.
func (DiscardHandler) OnNodeSkipped(printed *sync.Map, desc ocispec.Descriptor) error {
	return nil
}
