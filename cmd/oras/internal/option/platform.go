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
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Platform option struct.
type Platform struct {
	platform        []string
	Platform        *ocispec.Platform
	Platforms       []*ocispec.Platform
	FlagDescription string
}

// ApplyFlags applies flags to a command flag set.
func (opts *Platform) ApplyFlags(fs *pflag.FlagSet) {
	if opts.FlagDescription == "" {
		opts.FlagDescription = "request platform"
	}
	fs.StringSliceVarP(&opts.platform, "platform", "", nil, opts.FlagDescription+" in the form of `os[/arch][/variant][:os_version]` or comma-separated list for multiple platforms (supported in oras cp only)")
}

// Parse parses the input platform flag to an oci platform type.
func (opts *Platform) Parse(*cobra.Command) error {
	if len(opts.platform) == 0 {
		return nil
	}
	return opts.parsePlatform(opts.platform)
}

// parsePlatform parses multiple platforms
func (opts *Platform) parsePlatform(platformStrings []string) error {
	opts.Platforms = make([]*ocispec.Platform, 0, len(platformStrings))
	for _, platformStr := range platformStrings {
		platformStr = strings.TrimSpace(platformStr)
		if platformStr == "" {
			continue
		}
		var p ocispec.Platform
		platformPart, osVersion, _ := strings.Cut(platformStr, ":")
		parts := strings.Split(platformPart, "/")
		switch len(parts) {
		case 3:
			p.Variant = parts[2]
			fallthrough
		case 2:
			p.Architecture = parts[1]
		case 1:
			p.Architecture = runtime.GOARCH
		default:
			return fmt.Errorf("failed to parse platform %q: expected format os[/arch[/variant]]", platformStr)
		}
		p.OS = parts[0]
		if p.OS == "" {
			return fmt.Errorf("invalid platform: OS cannot be empty")
		}
		if p.Architecture == "" {
			return fmt.Errorf("invalid platform: Architecture cannot be empty")
		}
		p.OSVersion = osVersion
		opts.Platforms = append(opts.Platforms, &p)
	}

	// Set the first platform as the primary one for backward compatibility
	if len(opts.Platforms) > 0 {
		opts.Platform = opts.Platforms[0]
	}
	return nil
}

// ArtifactPlatform option struct.
type ArtifactPlatform struct {
	Platform
}

// ApplyFlags applies flags to a command flag set.
func (opts *ArtifactPlatform) ApplyFlags(fs *pflag.FlagSet) {
	opts.FlagDescription = "set artifact platform"
	fs.StringSliceVarP(&opts.platform, "artifact-platform", "", nil, "[Experimental] "+opts.FlagDescription+" in the form of `os[/arch][/variant][:os_version]`")
}
