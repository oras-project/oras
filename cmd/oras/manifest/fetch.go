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
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

type fetchOptions struct {
	option.Cache
	option.Common
	option.Descriptor
	option.Remote
	option.Platform
	option.Pretty

	mediaTypes []string
	outputPath string
	targetRef  string
}

func fetchCmd() *cobra.Command {
	var opts fetchOptions
	cmd := &cobra.Command{
		Use:   "fetch [flags] <name>{:<tag>|@<digest>}",
		Short: "[Preview] Fetch manifest of the target artifact",
		Long: `[Preview] Fetch manifest of the target artifact

** This command is in preview and under development. **

Example - Fetch raw manifest:
  oras manifest fetch localhost:5000/hello:latest

Example - Fetch the descriptor of a manifest:
  oras manifest fetch --descriptor localhost:5000/hello:latest

Example - Fetch manifest with specified media type:
  oras manifest fetch --media-type 'application/vnd.oci.image.manifest.v1+json' localhost:5000/hello:latest

Example - Fetch manifest with certain platform:
  oras manifest fetch --platform 'linux/arm/v5' localhost:5000/hello:latest

Example - Fetch manifest with prettified json result:
  oras manifest fetch --pretty localhost:5000/hello:latest
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.outputPath == "-" && opts.OutputDescriptor {
				return errors.New("`--output -` cannot be used with `--descriptor` at the same time")
			}

			return opts.ReadPassword()
		},
		Aliases: []string{"get"},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return fetchManifest(opts)
		},
	}

	cmd.Flags().StringSliceVarP(&opts.mediaTypes, "media-type", "", nil, "accepted media types")
	cmd.Flags().StringVarP(&opts.outputPath, "output", "o", "", "file `path` to write the fetched manifest to, use - for stdout")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func fetchManifest(opts fetchOptions) (fetchErr error) {
	ctx, _ := opts.SetLoggerLevel()

	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}

	if repo.Reference.Reference == "" {
		return oerrors.NewErrInvalidReference(repo.Reference)
	}
	repo.ManifestMediaTypes = opts.mediaTypes

	targetPlatform, err := opts.Parse()
	if err != nil {
		return err
	}

	src, err := opts.CachedTarget(repo)
	if err != nil {
		return err
	}

	var desc ocispec.Descriptor
	if opts.OutputDescriptor && opts.outputPath == "" {
		// fetch manifest descriptor only
		fetchOpts := oras.DefaultResolveOptions
		fetchOpts.TargetPlatform = targetPlatform
		desc, err = oras.Resolve(ctx, src, opts.targetRef, fetchOpts)
		if err != nil {
			return err
		}
	} else {
		// fetch manifest content
		var content []byte
		fetchOpts := oras.DefaultFetchBytesOptions
		fetchOpts.TargetPlatform = targetPlatform
		desc, content, err = oras.FetchBytes(ctx, src, opts.targetRef, fetchOpts)
		if err != nil {
			return err
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
