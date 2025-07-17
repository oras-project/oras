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
)

// Prompts for pull events.
const (
	PullPromptDownloading = "Downloading"
	PullPromptPulled      = "Pulled     "
	PullPromptProcessing  = "Processing "
	PullPromptSkipped     = "Skipped    "
	PullPromptRestored    = "Restored   "
	PullPromptDownloaded  = "Downloaded "
)

// Prompts for push/attach events.
const (
	PushPromptUploaded  = "Uploaded "
	PushPromptUploading = "Uploading"
	PushPromptSkipped   = "Skipped  "
	PushPromptExists    = "Exists   "
)

// Prompts for cp events.
const (
	copyPromptExists  = "Exists "
	copyPromptCopying = "Copying"
	copyPromptCopied  = "Copied "
	copyPromptSkipped = "Skipped"
	copyPromptMounted = "Mounted"
)

// Prompts for backup events.
const (
	backupPromptPulling = "Pulling  "
	backupPromptPulled  = "Pulled   "
	backupPromptExists  = "Exists   "
	backupPromptSkipped = "Skipped  "
)

// Prompts for index events.
const (
	IndexPromptFetching = "Fetching "
	IndexPromptFetched  = "Fetched  "
	IndexPromptAdded    = "Added    "
	IndexPromptMerged   = "Merged   "
	IndexPromptRemoved  = "Removed  "
	IndexPromptPacked   = "Packed   "
	IndexPromptPushed   = "Pushed   "
	IndexPromptUpdated  = "Updated  "
)

// DeduplicatedFilter filters out deduplicated descriptors.
func DeduplicatedFilter(committed *sync.Map) func(desc ocispec.Descriptor) bool {
	return func(desc ocispec.Descriptor) bool {
		name := desc.Annotations[ocispec.AnnotationTitle]
		v, ok := committed.Load(desc.Digest.String())
		// committed but not printed == deduplicated
		return ok && v != name
	}
}
