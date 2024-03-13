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

// PushHandler handles status output for push command.
type PushHandler interface {
	OnFileLoading(name string) error
	OnEmptyArtifact() error
	TrackTarget(gt oras.GraphTarget) (oras.GraphTarget, error)
	UpdatePushCopyOptions(opts *oras.CopyGraphOptions, fetcher content.Fetcher)
}

// AttachHandler handles text status output for attach command.
type AttachHandler PushHandler

// PullHandler handles status output for pull command.
type PullHandler interface {
	TrackTarget(gt oras.GraphTarget) (oras.GraphTarget, error)
	StopTracking()
	OnNodeProcessing(desc ocispec.Descriptor) error
	OnNodeDownloading(desc ocispec.Descriptor) error
	OnNodeSkipped(printed *sync.Map, desc ocispec.Descriptor) error
	UpdatePullCopyOptions(opts *oras.CopyGraphOptions, printed *sync.Map, includeSubject bool, configPath string, configMediaType string)
}
