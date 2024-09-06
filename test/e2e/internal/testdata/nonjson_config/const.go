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

package nonjson_config

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
	Descriptor = ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:9d16f5505246424aed7116cb21216704ba8c919997d0f1f37e154c11d509e1d2",
		Size:      529,
	}
)
