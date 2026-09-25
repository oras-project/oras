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
	platform        string
	Platform        *ocispec.Platform
	FlagDescription string
}

// ApplyFlags applies flags to a command flag set.
func (opts *Platform) ApplyFlags(fs *pflag.FlagSet) {
	if opts.FlagDescription == "" {
		opts.FlagDescription = "request platform"
	}
	fs.StringVarP(&opts.platform, "platform", "", "", opts.FlagDescription+" in the form of `os[/arch][/variant][:os_version]`")
}

// Parse parses the input platform flag to an oci platform type.
func (opts *Platform) Parse(*cobra.Command) error {
	if opts.platform == "" {
		return nil
	}
	p, err := parsePlatform(opts.platform)
	if err != nil {
		return err
	}
	opts.Platform = p
	return nil
}

// Platforms is the multi-value platform option used by commands that support
// selecting more than one platform at a time.
type Platforms struct {
	platforms       []string
	Platforms       []*ocispec.Platform
	FlagDescription string
}

// ApplyFlags applies the --platform flag to a multi-platform option.
func (opts *Platforms) ApplyFlags(fs *pflag.FlagSet) {
	if opts.FlagDescription == "" {
		opts.FlagDescription = "request platform"
	}
	fs.StringSliceVarP(&opts.platforms, "platform", "", nil, opts.FlagDescription+" in the form of `os[/arch][/variant][:os_version]` or a comma-separated list")
}

// Parse parses the input platform flags to OCI platform types.
func (opts *Platforms) Parse(*cobra.Command) error {
	if len(opts.platforms) == 0 {
		return nil
	}
	opts.Platforms = make([]*ocispec.Platform, 0, len(opts.platforms))
	seen := make(map[string]struct{}, len(opts.platforms))
	for _, platformStr := range opts.platforms {
		platformStr = strings.TrimSpace(platformStr)
		if platformStr == "" {
			return fmt.Errorf("invalid platform: value cannot be empty")
		}
		p, err := parsePlatform(platformStr)
		if err != nil {
			return err
		}
		key := strings.Join([]string{p.OS, p.Architecture, p.Variant, p.OSVersion}, "\x00")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		opts.Platforms = append(opts.Platforms, p)
	}
	return nil
}

func parsePlatform(platformStr string) (*ocispec.Platform, error) {
	// OS[/Arch[/Variant]][:OSVersion]
	// If Arch is not provided, use GOARCH instead.
	var p ocispec.Platform
	platformPart, osVersion, _ := strings.Cut(platformStr, ":")
	p.OSVersion = osVersion
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
		return nil, fmt.Errorf("failed to parse platform %q: expected format os[/arch[/variant]]", platformStr)
	}
	p.OS = parts[0]
	if p.OS == "" {
		return nil, fmt.Errorf("invalid platform: OS cannot be empty")
	}
	if p.Architecture == "" {
		return nil, fmt.Errorf("invalid platform: Architecture cannot be empty")
	}
	return &p, nil
}

// ArtifactPlatform option struct.
type ArtifactPlatform struct {
	Platform
}

// ApplyFlags applies flags to a command flag set.
func (opts *ArtifactPlatform) ApplyFlags(fs *pflag.FlagSet) {
	opts.FlagDescription = "set artifact platform"
	fs.StringVarP(&opts.platform, "artifact-platform", "", "", "[Experimental] "+opts.FlagDescription+" in the form of `os[/arch][/variant][:os_version]`")
}
