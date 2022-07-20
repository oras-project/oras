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
)

// Platform option struct.
type Platform struct {
	Platform string
}

// ApplyFlags applies flags to a command flag set.
func (opts *Platform) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&opts.Platform, "platform", "", "", "fetch the manifest of a specific platform if target is multi-platform capable")
}

// ParsePlatform parses the input platform flag to an oci platform type.
func (opts *Platform) ParsePlatform() (ocispec.Platform, error) {
	var p ocispec.Platform
	parts := strings.SplitN(opts.Platform, ":", 2)
	if len(parts) == 2 {
		// OSVersion is splitted by comma
		p.OSVersion = parts[1]
	}

	parts = strings.Split(parts[0], "/")
	if len(parts) < 2 {
		return ocispec.Platform{}, fmt.Errorf("failed to parse platform '%s': expected format os/arch[/variant]", opts.Platform)
	}
	if len(parts) > 3 {
		return ocispec.Platform{}, fmt.Errorf("failed to parse platform '%s': too many slashes", opts.Platform)
	}

	// OS/Arch/[Variant]
	p.OS = parts[0]
	p.Architecture = parts[1]
	if len(parts) > 2 {
		p.Variant = parts[2]
	}

	return p, nil
}

// FetchManifest fetches the manifest content of reference from target.
// If platform flag not empty, will fetch the specified platform.
func (opts *Platform) FetchManifest(ctx context.Context, target oras.Target, reference string) ([]byte, error) {
	desc, manifest, err := fetchAndVerify(ctx, target, reference)
	if err != nil {
		return nil, err
	}
	if opts.Platform != "" {
		// TODO: replace this with oras-go support when oras-project/oras-go#210 is done
		if desc.MediaType != ocispec.MediaTypeImageIndex && desc.MediaType != "application/vnd.docker.distribution.manifest.list.v2+json" {
			return nil, fmt.Errorf("%q is not a multi-platform media type", desc.MediaType)
		}
		return opts.fetchPlatform(ctx, target, manifest)
	}
	return manifest, nil
}

// TODO: replace this with oras-go support when oras-project/oras-go#210 is done
func (opts *Platform) fetchPlatform(ctx context.Context, fetcher content.Fetcher, root []byte) ([]byte, error) {
	target, err := opts.ParsePlatform()
	if err != nil {
		return nil, err
	}

	var index ocispec.Index
	if err := json.Unmarshal(root, &index); err != nil {
		return nil, err
	}

	for _, m := range index.Manifests {
		if target.OS == m.Platform.OS &&
			target.Architecture == m.Platform.Architecture &&
			(target.Variant == "" || target.Variant == m.Platform.Variant) &&
			(target.OSVersion == "" || target.OSVersion == m.Platform.OSVersion) {
			return content.FetchAll(ctx, fetcher, m)
		}
	}
	return nil, fmt.Errorf("failed to find platform matching the flag %q", opts.Platform)
}

func fetchAndVerify(ctx context.Context, target oras.Target, reference string) (ocispec.Descriptor, []byte, error) {
	// TODO: replace this when oras-project/oras-go#102 is done
	// Read and verify digest
	desc, err := target.Resolve(ctx, reference)
	if err != nil {
		return ocispec.Descriptor{}, nil, err
	}
	manifest, err := content.FetchAll(ctx, target, desc)
	return desc, manifest, err
}
