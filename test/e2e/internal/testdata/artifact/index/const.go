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

package index

import "oras.land/oras/test/e2e/internal/utils/match"

var (
	ManifestDigest    = "sha256:7679bc22c33b87aa345c6950a993db98a6df7a6cc77a35c388908a3a50be6bad"
	ManifestStatusKey = match.StateKey{
		Digest: "7679bc22c33b",
		Name:   "application/vnd.oci.image.index.v1+json",
	}
)
