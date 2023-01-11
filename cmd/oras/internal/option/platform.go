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
	"runtime"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/pflag"
)

// Platform option struct.
type Platform struct {
	platform string
	Platform *ocispec.Platform
}

// ApplyFlags applies flags to a command flag set.
func (opts *Platform) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&opts.platform, "platform", "", "", "request platform in the form of `os[/arch][/variant][:os_version]`")
}

// parse parses the input platform flag to an oci platform type.
func (opts *Platform) Parse() error {
	if opts.platform == "" {
		return nil
	}

	// OS[/Arch[/Variant]][:OSVersion]
	// If Arch is not provided, will use GOARCH instead
	var platformStr string
	var p ocispec.Platform
	platformStr, p.OSVersion, _ = strings.Cut(opts.platform, ":")
	parts := strings.Split(platformStr, "/")
	switch len(parts) {
	case 3:
		p.Variant = parts[2]
		fallthrough
	case 2:
		p.Architecture = parts[1]
	case 1:
		p.Architecture = runtime.GOARCH
	default:
		return fmt.Errorf("failed to parse platform %q: expected format os[/arch[/variant]]", opts.platform)
	}
	p.OS = parts[0]
	if p.OS == "" {
		return fmt.Errorf("invalid platform: OS cannot be empty")
	}
	if p.Architecture == "" {
		return fmt.Errorf("invalid platform: Architecture cannot be empty")
	}
	opts.Platform = &p
	return nil
}
