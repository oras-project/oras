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

package metadata

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/option"
)

// Renderer renders metadata information when an operation is complete.
type Renderer interface {
	Render() error
}

// PushHandler handles metadata output for push events.
type PushHandler interface {
	TaggedHandler
	Renderer

	OnCopied(opts *option.Target, root ocispec.Descriptor) error
}

// AttachHandler handles metadata output for attach events.
type AttachHandler interface {
	Renderer

	OnAttached(target *option.Target, root ocispec.Descriptor, subject ocispec.Descriptor)
}

// DiscoverHandler handles metadata output for discover events.
type DiscoverHandler interface {
	Renderer

	// MultiLevelSupported returns true if the handler supports multi-level
	// discovery.
	MultiLevelSupported() bool
	// OnDiscovered is called after a referrer is discovered.
	OnDiscovered(referrer, subject ocispec.Descriptor) error
}

// ManifestFetchHandler handles metadata output for manifest fetch events.
type ManifestFetchHandler interface {
	// OnFetched is called after the manifest content is fetched.
	OnFetched(path string, desc ocispec.Descriptor, content []byte) error
}

// PullHandler handles metadata output for pull events.
type PullHandler interface {
	Renderer

	// OnLayerSkipped is called when a layer is skipped.
	OnLayerSkipped(ocispec.Descriptor) error
	// OnFilePulled is called after a file is pulled.
	OnFilePulled(name string, outputDir string, desc ocispec.Descriptor, descPath string) error
	// OnPulled is called when a pull operation completes.
	OnPulled(target *option.Target, desc ocispec.Descriptor)
}

// TaggedHandler handles status output for tag command.
type TaggedHandler interface {
	// OnTagged is called when each tagging operation is done.
	OnTagged(desc ocispec.Descriptor, tag string) error
}

// TagHandler handles status output for tag command.
type TagHandler interface {
	// OnTagging is called when tagging starts.
	OnTagging(desc ocispec.Descriptor, tag string) error
	TaggedHandler
}

// ManifestPushHandler handles metadata output for manifest push events.
type ManifestPushHandler interface {
	TaggedHandler
	Renderer

	OnManifestPushed(desc ocispec.Descriptor) error
}

// ManifestIndexCreateHandler handles metadata output for index create events.
type ManifestIndexCreateHandler interface {
	TaggedHandler
	Renderer

	OnIndexCreated(desc ocispec.Descriptor)
}

// ManifestIndexUpdateHandler handles metadata output for index update events.
type ManifestIndexUpdateHandler ManifestIndexCreateHandler

// CopyHandler handles metadata output for cp events.
type CopyHandler interface {
	TaggedHandler
}
