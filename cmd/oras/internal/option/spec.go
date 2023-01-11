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

package option

import (
	"fmt"

	"github.com/spf13/pflag"
	"oras.land/oras/internal/registry"
)

// ImageSpec option struct.
type ImageSpec struct {
	ManifestSupportState registry.ManifestSupportState

	// should be provided in form of `<version>-<manifest type>`
	specFlag string
}

// Parse parses flags into the option.
func (opts *ImageSpec) Parse() error {
	switch opts.specFlag {
	case "":
		opts.ManifestSupportState = registry.ManifestSupportUnknown
	case "v1.1-image":
		opts.ManifestSupportState = registry.OCIImage
	case "v1.1-artifact":
		opts.ManifestSupportState = registry.OCIArtifact
	default:
		return fmt.Errorf("unknown image specification flag: %q", opts.specFlag)
	}
	return nil
}

// ApplyFlags applies flags to a command flag set.
func (opts *ImageSpec) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringVar(&opts.specFlag, "image-spec", "", "set OCI image spec version and request manifest type. E.g. `v1.1-image,v1.1-artifact`")
}

// DistributionSpec option struct.
type DistributionSpec struct {
	ReferrersApiSupportState registry.ReferrersApiSupportState

	// should be provided in form of`<version>-<api>-<option>`
	specFlag string
}

// Parse parses flags into the option.
func (opts *DistributionSpec) Parse() error {
	switch opts.specFlag {
	case "":
		opts.ReferrersApiSupportState = registry.ReferrersApiSupportUnknown
	case "v1.1-referrers-api":
		opts.ReferrersApiSupportState = registry.ReferrersApiUnsupported
	case "v1.1-referrers-tag":
		opts.ReferrersApiSupportState = registry.ReferrersApiSupported
	default:
		return fmt.Errorf("unknown image specification flag: %q", opts.specFlag)
	}
	return nil
}

// ApplyFlags applies flags to a command flag set.
func (opts *DistributionSpec) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringVar(&opts.specFlag, "distribution-spec", "", "set OCI distribution spec version and API option. E.g. `v1.1-referrers-api,v1.1-referrers-tag`")
}
