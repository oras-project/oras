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
	"strings"

	"github.com/spf13/pflag"
	"oras.land/oras-go/v2"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
)

const (
	ImageSpecV1_1                    = "v1.1"
	ImageSpecV1_0                    = "v1.0"
	DistributionSpecReferrersTagV1_1 = "v1.1-referrers-tag"
	ReferrersAPIV1_1                 = "v1.1-referrers-api"
)

// ImageSpec option struct which implements pflag.Value interface.
type ImageSpec struct {
	flag        string
	PackVersion oras.PackManifestVersion
}

// Set validates and sets the flag value from a string argument.
func (is *ImageSpec) Set(value string) error {
	is.flag = value
	switch value {
	case ImageSpecV1_1:
		is.PackVersion = oras.PackManifestVersion1_1_RC4
	case ImageSpecV1_0:
		is.PackVersion = oras.PackManifestVersion1_0
	default:
		return &oerrors.Error{
			Err:            fmt.Errorf("unknown image specification flag: %s", value),
			Recommendation: fmt.Sprintf("Available options: %s", is.Type()),
		}
	}
	return nil
}

// Type returns the string value of the inner flag.
func (is *ImageSpec) Type() string {
	return strings.Join([]string{ImageSpecV1_1, ImageSpecV1_0}, ",")
}

// String returns the string representation of the flag.
func (is *ImageSpec) String() string {
	return is.flag
}

// ApplyFlags applies flags to a command flag set.
func (is *ImageSpec) ApplyFlags(fs *pflag.FlagSet) {
	// default to v1.1-rc.4
	is.PackVersion = oras.PackManifestVersion1_1_RC4
	fs.Var(is, "image-spec", `[Experimental] specify manifest type for building artifact (default "v1.1")`)
}

// DistributionSpec option struct which implements pflag.Value interface.
type DistributionSpec struct {
	// ReferrersAPI indicates the preference of the implementation of the Referrers API.
	// Set to true for referrers API, false for referrers tag scheme, and nil for auto fallback.
	ReferrersAPI *bool

	// specFlag should be provided in form of`<version>-<api>-<option>`
	flag string
}

// Set validates and sets the flag value from a string argument.
func (ds *DistributionSpec) Set(value string) error {
	ds.flag = value
	switch ds.flag {
	case DistributionSpecReferrersTagV1_1:
		isApi := false
		ds.ReferrersAPI = &isApi
	case ReferrersAPIV1_1:
		isApi := true
		ds.ReferrersAPI = &isApi
	default:
		return &oerrors.Error{
			Err:            fmt.Errorf("unknown distribution specification flag: %s", value),
			Recommendation: fmt.Sprintf("Available options: %s", ds.Type()),
		}
	}
	return nil
}

// Type returns the string value of the inner flag.
func (ds *DistributionSpec) Type() string {
	return strings.Join([]string{DistributionSpecReferrersTagV1_1, ReferrersAPIV1_1}, ",")
}

// String returns the string representation of the flag.
func (ds *DistributionSpec) String() string {
	return ds.flag
}

// ApplyFlagsWithPrefix applies flags to a command flag set with a prefix string.
func (ds *DistributionSpec) ApplyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	flagPrefix, notePrefix := applyPrefix(prefix, description)
	fs.Var(ds, flagPrefix+"distribution-spec", "[Preview] set OCI distribution spec version and API option for "+notePrefix+"target.")
}
