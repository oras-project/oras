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
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
)

// PushHandler handles status output for push command.
type PushHandler interface {
	OnFileLoading(name string) error
	OnEmptyArtifact() error
	TrackTarget(gt oras.GraphTarget) (oras.GraphTarget, error)
	UpdateCopyOptions(opts *oras.CopyGraphOptions, fetcher content.Fetcher)
}

// AttachHandler handles text status output for attach command.
type AttachHandler PushHandler

// PullHandler handles status output for pull command.
type PullHandler interface {
	// TrackTarget returns a tracked target.
	// If no TTY is available, it returns the original target.
	TrackTarget(gt oras.GraphTarget) (oras.GraphTarget, error)
	// OnNodeProcessing is called when processing a manifest.
	OnNodeProcessing(desc ocispec.Descriptor) error
	// OnNodeDownloading is called before downloading a node.
	OnNodeDownloading(desc ocispec.Descriptor) error
	// OnNodeDownloaded is called after a node is downloaded.
	OnNodeDownloaded(desc ocispec.Descriptor) error
	// OnNodeRestored is called after a deduplicated node is restored.
	OnNodeRestored(desc ocispec.Descriptor) error
	// OnNodeSkipped is called when a node is skipped.
	OnNodeSkipped(desc ocispec.Descriptor) error
	io.Closer
}
