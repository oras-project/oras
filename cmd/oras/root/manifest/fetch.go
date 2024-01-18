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

package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras/cmd/oras/internal/argument"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

type fetchOptions struct {
	option.Cache
	option.Common
	option.Descriptor
	option.Platform
	option.Pretty
	option.Target

	mediaTypes []string
	outputPath string
}

func fetchCmd() *cobra.Command {
	var opts fetchOptions
	cmd := &cobra.Command{
		Use:   "fetch [flags] <name>{:<tag>|@<digest>}",
		Short: "Fetch manifest of the target artifact",
		Long: `Fetch manifest of the target artifact

Example - Fetch raw manifest from a registry:
  oras manifest fetch localhost:5000/hello:v1

Example - Fetch the descriptor of a manifest from a registry:
  oras manifest fetch --descriptor localhost:5000/hello:v1

Example - Fetch manifest from a registry with specified media type:
  oras manifest fetch --media-type 'application/vnd.oci.image.manifest.v1+json' localhost:5000/hello:v1

Example - Fetch manifest from a registry with certain platform:
  oras manifest fetch --platform 'linux/arm/v5' localhost:5000/hello:v1

Example - Fetch manifest from a registry with prettified json result:
  oras manifest fetch --pretty localhost:5000/hello:v1

Example - Fetch raw manifest from an OCI image layout folder 'layout-dir':
  oras manifest fetch --oci-layout layout-dir:v1

Example - Fetch raw manifest from an OCI layout archive file 'layout.tar':
  oras manifest fetch --oci-layout layout.tar:v1
`,
		Args: oerrors.CheckArgs(argument.Exactly(1), "the manifest to fetch"),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.outputPath == "-" && opts.OutputDescriptor {
				return errors.New("`--output -` cannot be used with `--descriptor` at the same time")
			}
			opts.RawReference = args[0]
			return option.Parse(&opts)
		},
		Aliases: []string{"get"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return fetchManifest(cmd, &opts)
		},
	}

	cmd.Flags().StringSliceVarP(&opts.mediaTypes, "media-type", "", nil, "accepted media types")
	cmd.Flags().StringVarP(&opts.outputPath, "output", "o", "", "file `path` to write the fetched manifest to, use - for stdout")
	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.Target)
}

func fetchManifest(cmd *cobra.Command, opts *fetchOptions) (fetchErr error) {
	ctx, logger := opts.WithContext(cmd.Context())

	target, err := opts.NewReadonlyTarget(ctx, opts.Common, logger)
	if err != nil {
		return err
	}
	if err := opts.EnsureReferenceNotEmpty(cmd, true); err != nil {
		return err
	}
	if repo, ok := target.(*remote.Repository); ok {
		repo.ManifestMediaTypes = opts.mediaTypes
	} else if opts.mediaTypes != nil {
		return fmt.Errorf("`--media-type` cannot be used with `--oci-layout` at the same time")
	}

	src, err := opts.CachedTarget(target)
	if err != nil {
		return err
	}

	var desc ocispec.Descriptor
	if opts.OutputDescriptor && opts.outputPath == "" {
		// fetch manifest descriptor only
		fetchOpts := oras.DefaultResolveOptions
		fetchOpts.TargetPlatform = opts.Platform.Platform
		desc, err = oras.Resolve(ctx, src, opts.Reference, fetchOpts)
		if err != nil {
			return fmt.Errorf("failed to find %q: %w", opts.RawReference, err)
		}
	} else {
		// fetch manifest content
		var content []byte
		fetchOpts := oras.DefaultFetchBytesOptions
		fetchOpts.TargetPlatform = opts.Platform.Platform
		desc, content, err = oras.FetchBytes(ctx, src, opts.Reference, fetchOpts)
		if err != nil {
			return fmt.Errorf("failed to fetch the content of %q: %w", opts.RawReference, err)
		}

		if opts.outputPath == "" || opts.outputPath == "-" {
			// output manifest content
			return opts.Output(os.Stdout, content)
		}

		// save manifest content into the local file if the output path is provided
		if err = os.WriteFile(opts.outputPath, content, 0666); err != nil {
			return err
		}
	}

	// output manifest's descriptor if `--descriptor` is used
	if opts.OutputDescriptor {
		descBytes, err := json.Marshal(desc)
		if err != nil {
			return err
		}
		return opts.Output(os.Stdout, descBytes)
	}

	return nil
}
