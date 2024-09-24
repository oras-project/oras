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

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
)

func discardStopTrack() error {
	return nil
}

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

// TrackTarget returns a target with status tracking.
func (DiscardHandler) TrackTarget(gt oras.GraphTarget) (oras.GraphTarget, StopTrackTargetFunc, error) {
	return gt, discardStopTrack, nil
}

// OnCopySkipped is called when an object already exists.
func (DiscardHandler) OnCopySkipped(_ context.Context, _ ocispec.Descriptor) error {
	return nil
}

// PreCopy implements PreCopy of CopyHandler.
func (DiscardHandler) PreCopy(_ context.Context, _ ocispec.Descriptor) error {
	return nil
}

// PostCopy implements PostCopy of CopyHandler.
func (DiscardHandler) PostCopy(_ context.Context, _ ocispec.Descriptor) error {
	return nil
}

// OnNodeDownloading implements PullHandler.
func (DiscardHandler) OnNodeDownloading(desc ocispec.Descriptor) error {
	return nil
}

// OnNodeDownloaded implements PullHandler.
func (DiscardHandler) OnNodeDownloaded(desc ocispec.Descriptor) error {
	return nil
}

// OnNodeRestored implements PullHandler.
func (DiscardHandler) OnNodeRestored(_ ocispec.Descriptor) error {
	return nil
}

// OnNodeProcessing implements PullHandler.
func (DiscardHandler) OnNodeProcessing(desc ocispec.Descriptor) error {
	return nil
}

// OnNodeProcessing implements PullHandler.
func (DiscardHandler) OnNodeSkipped(desc ocispec.Descriptor) error {
	return nil
}

// OnSourceManifestFetching implements ManifestIndexCreateHandler.
func (DiscardHandler) OnSourceManifestFetching(source string) error {
	return nil
}

// OnSourceManifestFetched implements ManifestIndexCreateHandler.
func (DiscardHandler) OnSourceManifestFetched(source string) error {
	return nil
}

// OnManifestFetching implements ManifestIndexUpdateHandler.
func (DiscardHandler) OnManifestFetching(ref string) error {
	return nil
}

// OnManifestFetched implements ManifestIndexUpdateHandler.
func (DiscardHandler) OnManifestFetched(ref string, digest digest.Digest) error {
	return nil
}

// OnManifestRemoved implements ManifestIndexUpdateHandler.
func (DiscardHandler) OnManifestRemoved(digest digest.Digest) error {
	return nil
}

// OnManifestAdded implements ManifestIndexUpdateHandler.
func (DiscardHandler) OnManifestAdded(ref string, digest digest.Digest) error {
	return nil
}

// OnIndexFetching implements ManifestIndexUpdateHandler.
func (DiscardHandler) OnIndexFetching(indexRef string) error {
	return nil
}

// OnIndexFetched implements ManifestIndexUpdateHandler.
func (DiscardHandler) OnIndexFetched(indexRef string, digest digest.Digest) error {
	return nil
}

// OnIndexMerged implements ManifestIndexUpdateHandler.
func (DiscardHandler) OnIndexMerged(indexRef string, digest digest.Digest) error {
	return nil
}

// OnIndexPacked implements ManifestIndexCreateHandler.
func (DiscardHandler) OnIndexPacked(shortDigest string) error {
	return nil
}

// OnIndexUpdated implements ManifestIndexUpdateHandler.
func (DiscardHandler) OnIndexUpdated(digest digest.Digest) error {
	return nil
}

// OnIndexPushed implements ManifestIndexCreateHandler.
func (DiscardHandler) OnIndexPushed(path string) error {
	return nil
}
