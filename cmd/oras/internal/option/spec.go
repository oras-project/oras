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
)

const (
	V1_1 = "1.1"
	V1_0 = "1.0"
)

// ImageSpec option struct.
type ImageSpec struct {
	Flag string
}

// Parse parses flags into the option.
func (opts *ImageSpec) Parse() error {
	switch opts.Flag {
	case V1_1, V1_0:
		return nil
	}
	return fmt.Errorf("unknown image specification flag: %q", opts.Flag)
}

// ApplyFlags applies flags to a command flag set.
func (opts *ImageSpec) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringVar(&opts.Flag, "image-spec", V1_1, fmt.Sprintf("[Experimental] specify manifest type for building artifact. options: %s, %s", V1_1, V1_0))
}

// distributionSpec option struct.
type distributionSpec struct {
	// referrersAPI indicates the preference of the implementation of the Referrers API.
	// Set to true for referrers API, false for referrers tag scheme, and nil for auto fallback.
	referrersAPI *bool

	// specFlag should be provided in form of`<version>-<api>-<option>`
	specFlag string
}

// Parse parses flags into the option.
func (opts *distributionSpec) Parse() error {
	switch opts.specFlag {
	case "":
		opts.referrersAPI = nil
	case "v1.1-referrers-tag":
		isApi := false
		opts.referrersAPI = &isApi
	case "v1.1-referrers-api":
		isApi := true
		opts.referrersAPI = &isApi
	default:
		return fmt.Errorf("unknown distribution specification flag: %q", opts.specFlag)
	}
	return nil
}

// ApplyFlagsWithPrefix applies flags to a command flag set with a prefix string.
func (opts *distributionSpec) ApplyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	flagPrefix, notePrefix := applyPrefix(prefix, description)
	fs.StringVar(&opts.specFlag, flagPrefix+"distribution-spec", "", "[Preview] set OCI distribution spec version and API option for "+notePrefix+"target. options: v1.1-referrers-api, v1.1-referrers-tag")
}
