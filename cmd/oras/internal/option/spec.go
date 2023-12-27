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
	"oras.land/oras-go/v2"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
)

const (
	ImageSpecV1_1    = "v1.1"
	ImageSpecV1_0    = "v1.0"
	ReferrersTagV1_1 = "v1.1-referrers-tag"
	ReferrersAPIV1_1 = "v1.1-referrers-api"
)

// ImageSpec option struct.
type ImageSpec struct {
	flag        string
	PackVersion oras.PackManifestVersion
}

// Set validates and sets the flag value from a string argument.
func (i *ImageSpec) Set(value string) error {
	i.flag = value
	switch value {
	case ImageSpecV1_1:
		i.PackVersion = oras.PackManifestVersion1_1_RC4
	case ImageSpecV1_0:
		i.PackVersion = oras.PackManifestVersion1_0
	default:
		return &oerrors.Error{
			Err:            fmt.Errorf("unknown image specification flag: %s", value),
			Recommendation: fmt.Sprintf("Available options: %s", i.Type()),
		}
	}
	return nil
}

// Type returns the type of the flag.
func (i *ImageSpec) Type() string {
	return fmt.Sprintf("%s, %s", ImageSpecV1_1, ImageSpecV1_0)
}

// String returns the string representation of the flag.
func (i *ImageSpec) String() string {
	return i.flag
}

// ApplyFlags applies flags to a command flag set.
func (opts *ImageSpec) ApplyFlags(fs *pflag.FlagSet) {
	// default to v1.1-rc.4
	opts.PackVersion = oras.PackManifestVersion1_1_RC4
	fs.Var(opts, "image-spec", "[Experimental] specify manifest type for building artifact")
}

// DistributionSpec option struct.
type DistributionSpec struct {
	// ReferrersAPI indicates the preference of the implementation of the Referrers API.
	// Set to true for referrers API, false for referrers tag scheme, and nil for auto fallback.
	ReferrersAPI *bool

	// specFlag should be provided in form of`<version>-<api>-<option>`
	flag string
}

// Set validates and sets the flag value from a string argument.
func (d *DistributionSpec) Set(value string) error {
	d.flag = value
	switch d.flag {
	case ReferrersTagV1_1:
		isApi := false
		d.ReferrersAPI = &isApi
	case ReferrersAPIV1_1:
		isApi := true
		d.ReferrersAPI = &isApi
	default:
		return &oerrors.Error{
			Err:            fmt.Errorf("unknown distribution specification flag: %s", value),
			Recommendation: fmt.Sprintf("Available options: %s", d.Type()),
		}
	}
	return nil
}

// Type returns the type of the flag.
func (d *DistributionSpec) Type() string {
	return fmt.Sprintf("%s, %s", ReferrersTagV1_1, ReferrersAPIV1_1)
}

// String returns the string representation of the flag.
func (d *DistributionSpec) String() string {
	return d.flag
}

// ApplyFlagsWithPrefix applies flags to a command flag set with a prefix string.
func (opts *DistributionSpec) ApplyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	flagPrefix, notePrefix := applyPrefix(prefix, description)
	fs.Var(opts, flagPrefix+"distribution-spec", "[Preview] set OCI distribution spec version and API option for "+notePrefix+"target. options: v1.1-referrers-api, v1.1-referrers-tag")
}
