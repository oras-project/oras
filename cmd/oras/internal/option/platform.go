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
	"fmt"
	"io"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/pflag"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry/remote"
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
		// OSVersion is splitted by comma
		p.OSVersion = parts[1]
	}

	parts = strings.Split(parts[0], "/")
	if len(parts) < 2 || len(parts) > 3 {
		return ocispec.Platform{}, fmt.Errorf("failed to parse platform '%s': expected format os/arch[/variant]", opts.Platform)
	}

	// OS/Arch/[Variant]
	p.OS = parts[0]
	if p.OS == "" {
		return ocispec.Platform{}, fmt.Errorf("invalide platform: OS cannot be empty")
	}
	p.Architecture = parts[1]
	if p.Architecture == "" {
		return ocispec.Platform{}, fmt.Errorf("invalide platform: Architecture cannot be empty")
	}
	if len(parts) > 2 {
		p.Variant = parts[2]
	}

	return p, nil
}

// FetchManifest fetches the manifest content of reference from target.
// If platform flag not empty, will fetch the specified platform.
func (opts *Platform) FetchManifest(ctx context.Context, repo *remote.Repository, reference string) (manifest []byte, err error) {
	desc, rc, err := repo.FetchReference(ctx, reference)
	if err != nil {
		return
	}
	defer rc.Close()

	if opts.Platform != "" {
		if desc.MediaType != ocispec.MediaTypeImageIndex && desc.MediaType != "application/vnd.docker.distribution.manifest.list.v2+json" {
			return nil, fmt.Errorf("%q is not a multi-platform media type", desc.MediaType)
		}
		// TODO: replace this with oras-go support when oras-project/oras-go#210 is done
		if desc, rc, err = opts.fetchPlatform(ctx, repo, desc); err != nil {
			return
		}
		defer rc.Close()
	}
	return content.ReadAll(rc, desc)
}

// TODO: replace this with oras-go support when oras-project/oras-go#210 is done
func (opts *Platform) fetchPlatform(ctx context.Context, repo *remote.Repository, root ocispec.Descriptor) (empty ocispec.Descriptor, rc io.ReadCloser, err error) {
	want, err := opts.parse()
	if err != nil {
		return
	}

	manifests, err := content.Successors(ctx, repo, root)
	if err != nil {
		return
	}

	for _, got := range manifests {
		// TODO: Platform.OSFeatures is ignored
		if want.OS == got.Platform.OS &&
			want.Architecture == got.Platform.Architecture &&
			(want.Variant == "" || want.Variant == got.Platform.Variant) &&
			(want.OSVersion == "" || want.OSVersion == got.Platform.OSVersion) {
			rc, err := repo.Fetch(ctx, got)
			return got, rc, err
		}
	}
	return empty, nil, fmt.Errorf("failed to find platform matching the flag %q", opts.Platform)
}
