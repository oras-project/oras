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

package artifact

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/test/e2e/internal/utils/match"
)

const (
	DefaultArtifactType    = "application/vnd.unknown.artifact.v1"
	DefaultConfigMediaType = "application/vnd.oci.empty.v1+json"
)

var (
	DefaultConfigStateKey = match.StateKey{Digest: "44136fa355b3", Name: "application/vnd.oci.empty.v1+json"}
	EmptyLayerJSON        = ocispec.Descriptor{
		MediaType: "application/vnd.oci.empty.v1+json",
		Digest:    "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
		Size:      2,
		Data:      []byte(`{}`),
	}
)
