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

var (
	Flags = struct {
		Layout           string
		FromLayout       string
		ToLayout         string
		DistributionSpec string
		ImageSpec        string
	}{
		"--oci-layout",
		"--from-oci-layout",
		"--to-oci-layout",
		"--distribution-spec",
		"--image-spec",
	}
	RegistryErrorPrefix = "Error response from registry: "
	EmptyBodyPrefix     = "recognizable error message not found: "
	InvalidTag          = "i-dont-think-this-tag-exists"
)
