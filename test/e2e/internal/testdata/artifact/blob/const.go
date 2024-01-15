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

package blob

import "oras.land/oras/test/e2e/internal/utils/match"

const (
	Tag = "blob"
)

var (
	StateKeys = []match.StateKey{
		{Digest: "44136fa355b3", Name: "application/vnd.oci.empty.v1+json"},
		{Digest: "2ef548696ac7", Name: "hello.tar"},
		{Digest: "ccd5f15fb7e0", Name: "application/vnd.oci.image.manifest.v1+json"},
	}
)
