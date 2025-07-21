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
	Renderer

	OnCopied(target *option.BinaryTarget, desc ocispec.Descriptor) error
}

// BlobPushHandler handles metadata output for backup events.
type BackupHandler interface {
	Renderer

	OnTagsFound(tags []string) error
	OnArtifactPulled(tag string, referrerCount int) error
	OnTarExporting(path string) error
	OnTarExported(path string, size int64) error
	OnBackupCompleted(tagsCount int, path string) error
}

// BlobPushHandler handles metadata output for blob push events.
type BlobPushHandler interface {
	Renderer

	OnBlobPushed(target *option.Target) error
}

// ResolveHandler handles metadata output for resolve events.
type ResolveHandler interface {
	OnResolved(desc ocispec.Descriptor) error
}

// ManifestDeleteHandler handles metadata output for manifest delete events.
type ManifestDeleteHandler interface {
	OnManifestMissing() error
	OnManifestDeleted() error
}

// BlobDeleteHandler handles metadata output for blob delete events.
type BlobDeleteHandler interface {
	OnBlobMissing() error
	OnBlobDeleted() error
}

// RepoTagsHandler handles metadata output for repo tags command.
type RepoTagsHandler interface {
	Renderer

	// OnTagListed is called for each tag that is listed.
	OnTagListed(tag string) error
}

// RepoListHandler handles metadata output for repo ls command.
type RepoListHandler interface {
	Renderer

	// OnRepositoryListed is called for each repository that is listed.
	OnRepositoryListed(repo string) error
}
