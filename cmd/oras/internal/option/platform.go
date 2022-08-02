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
	"context"
	"encoding/json"
	"fmt"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/pflag"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry"
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
func (opts *Platform) parse() (ocispec.Platform, error) {
	var p ocispec.Platform
	parts := strings.SplitN(opts.Platform, ":", 2)
	if len(parts) == 2 {
		// OSVersion is splitted by colon
		p.OSVersion = parts[1]
	}

	parts = strings.Split(parts[0], "/")
	if len(parts) < 2 || len(parts) > 3 {
		return ocispec.Platform{}, fmt.Errorf("failed to parse platform '%s': expected format os/arch[/variant]", opts.Platform)
	}

	// OS/Arch/[Variant]
	p.OS = parts[0]
	if p.OS == "" {
		return ocispec.Platform{}, fmt.Errorf("invalid platform: OS cannot be empty")
	}
	p.Architecture = parts[1]
	if p.Architecture == "" {
		return ocispec.Platform{}, fmt.Errorf("invalid platform: Architecture cannot be empty")
	}
	if len(parts) > 2 {
		p.Variant = parts[2]
	}

	return p, nil
}

// FetchDescriptor fetches a minimal descriptor of reference from target.
// If platform flag not empty, will fetch the specified platform.
func (opts *Platform) FetchDescriptor(ctx context.Context, repo registry.Repository, reference string) ([]byte, error) {
	ro, err := opts.resolveOption()
	if err != nil {
		return nil, err
	}
	desc, err := oras.Resolve(ctx, repo, reference, ro)
	if err != nil {
		return nil, err
	}
	return json.Marshal(ocispec.Descriptor{
		MediaType: desc.MediaType,
		Digest:    desc.Digest,
		Size:      desc.Size,
	})
}

// FetchManifest fetches the manifest content of reference from target.
// If platform flag not empty, will fetch the specified platform.
func (opts *Platform) FetchManifest(ctx context.Context, repo registry.Repository, reference string) ([]byte, error) {
	ro, err := opts.resolveOption()
	if err != nil {
		return nil, err
	}
	desc, err := oras.Resolve(ctx, repo, reference, ro)
	if err != nil {
		return nil, err
	}

	rc, err := repo.Fetch(ctx, desc)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return content.ReadAll(rc, desc)
}

func (opts *Platform) resolveOption() (ro oras.ResolveOptions, err error) {
	if opts.Platform != "" {
		var p ocispec.Platform
		if p, err = opts.parse(); err != nil {
			return
		}
		ro.TargetPlatform = &p
	}
	return
}
