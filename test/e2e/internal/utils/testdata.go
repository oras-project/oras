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

package utils

const (
	PreviewDesc      = "** This command is in preview and under development. **"
	ExampleDesc      = "\nExample - "
	ImageRepo        = "command/images"
	BlobRepo         = "command/blobs"
	ArtifactRepo     = "command/artifacts"
	Namespace        = "command"
	InvalidRepo      = "INVALID"
	LegacyConfigName = "legacy.registry.config"
	EmptyConfigName  = "empty.registry.config"
	// env
	RegHostKey         = "ORAS_REGISTRY_HOST"
	FallbackRegHostKey = "ORAS_REGISTRY_FALLBACK_HOST"
	ZOTHostKey         = "ZOT_REGISTRY_HOST"
)

var (
	TestDataRoot string
)
