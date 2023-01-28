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

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/pflag"
)

// ImageSpec option struct.
type ImageSpec struct {
	// Manifest type for building artifact
	ManifestMediaType string

	// specFlag should be provided in form of `<version>-<manifest type>`
	specFlag string
}

// Parse parses flags into the option.
func (opts *ImageSpec) Parse() error {
	switch opts.specFlag {
	case "":
		opts.ManifestMediaType = ""
	case "v1.1-image":
		opts.ManifestMediaType = ocispec.MediaTypeImageManifest
	case "v1.1-artifact":
		opts.ManifestMediaType = ocispec.MediaTypeArtifactManifest
	default:
		return fmt.Errorf("unknown image specification flag: %q", opts.specFlag)
	}
	return nil
}

// ApplyFlags applies flags to a command flag set.
func (opts *ImageSpec) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringVar(&opts.specFlag, "image-spec", "", "specify manifest type for building artifact. options: v1.1-image, v1.1-artifact")
}

// DistributionSpec option struct.
type DistributionSpec struct {
	// ReferrersAPI indicates the preference of the implementation of the Referrers API.
	// Set to true for referrers API, false for referrers tag scheme, and nil for auto fallback.
	ReferrersAPI *bool

	// specFlag should be provided in form of`<version>-<api>-<option>`
	specFlag string
}

// Parse parses flags into the option.
func (opts *DistributionSpec) Parse() error {
	switch opts.specFlag {
	case "":
		opts.ReferrersAPI = nil
	case "v1.1-referrers-tag":
		isApi := false
		opts.ReferrersAPI = &isApi
	case "v1.1-referrers-api":
		isApi := true
		opts.ReferrersAPI = &isApi
	default:
		return fmt.Errorf("unknown image specification flag: %q", opts.specFlag)
	}
	return nil
}

// ApplyFlagsWithPrefix applies flags to a command flag set with a prefix string.
func (opts *DistributionSpec) ApplyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	flagPrefix, notePrefix := applyPrefix(prefix, description)
	fs.StringVar(&opts.specFlag, flagPrefix+"distribution-spec", "", "set OCI distribution spec version and API option for "+notePrefix+"target. options: v1.1-referrers-api, v1.1-referrers-tag")
}
