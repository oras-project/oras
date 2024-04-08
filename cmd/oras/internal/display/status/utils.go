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

import ocispec "github.com/opencontainers/image-spec/specs-go/v1"

// GenerateContentKey generates a unique key for each content descriptor, using
// its digest and name if applicable.
func GenerateContentKey(desc ocispec.Descriptor) string {
	return desc.Digest.String() + desc.Annotations[ocispec.AnnotationTitle]
}

// Prompts for pull events.
const (
	PullPromptDownloading = "Downloading"
	PullPromptPulled      = "Pulled     "
	PullPromptProcessing  = "Processing "
	PullPromptSkipped     = "Skipped    "
	PullPromptRestored    = "Restored   "
	PullPromptDownloaded  = "Downloaded "
)
