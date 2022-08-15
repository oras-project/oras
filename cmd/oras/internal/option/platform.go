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

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/pflag"
)

// Platform option struct.
type Platform struct {
	Platform string
}

// ApplyFlags applies flags to a command flag set.
func (opts *Platform) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&opts.Platform, "platform", "", "", "fetch the manifest of a specific platform if target is multi-platform capable")
}

// parse parses the input platform flag to an oci platform type.
func (opts *Platform) Parse() (*ocispec.Platform, error) {
	var platformStr string
	var p ocispec.Platform

	// OS/Arch/[Variant][:OSVersion]
	platformStr, p.OSVersion, _ = strings.Cut(opts.Platform, ":")
	parts := strings.Split(platformStr, "/")
	if len(parts) < 2 || len(parts) > 3 {
		return nil, fmt.Errorf("failed to parse platform '%s': expected format os/arch[/variant]", opts.Platform)
	}
	p.OS = parts[0]
	if p.OS == "" {
		return nil, fmt.Errorf("invalid platform: OS cannot be empty")
	}
	p.Architecture = parts[1]
	if p.Architecture == "" {
		return nil, fmt.Errorf("invalid platform: Architecture cannot be empty")
	}
	if len(parts) == 3 {
		p.Variant = parts[2]
	}

	return &p, nil
}
