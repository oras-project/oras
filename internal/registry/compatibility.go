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

package registry

// ManifestSupportState represents the state of remote registry support for
// manifest type.
type ManifestSupportState = int32

const (
	// ManifestSupportUnknown represents an unknown state of Referrers API.
	ManifestSupportUnknown ManifestSupportState = iota
	// OCIArtifact means that the remote registry is known to support oci artifact manifest type.
	OCIArtifact
	// OCIImage means that the remote registry is known to
	// support oci image manifest type only.
	OCIImage
)

// ReferrersApiSupportState represents the state of remote registry support for
// referrers API.
type ReferrersApiSupportState = int32

const (
	// ReferrersApiSupportUnknown represents an unknown state of Referrers API.
	ReferrersApiSupportUnknown ReferrersApiSupportState = iota
	// ReferrersApiSupported means that the remote registry is known to supporting referrers API.
	ReferrersApiSupported
	// ReferrersApiUnsupported means that the remote registry is known to not supporting referrers API.
	ReferrersApiUnsupported
)
