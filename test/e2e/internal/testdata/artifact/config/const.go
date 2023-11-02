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

package config

import "oras.land/oras/test/e2e/internal/utils/match"

const (
	Tag = "config"
)

var (
	StateKeys = []match.StateKey{
		{Digest: "fe9dbc99451d", Name: "application/vnd.oci.image.config.v1+json"},
		{Digest: "2ef548696ac7", Name: "hello.tar"},
		{Digest: "0e7404d1cc4a", Name: "application/vnd.oci.image.manifest.v1+json"},
	}
)
