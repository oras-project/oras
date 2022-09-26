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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/descriptor"
)

type fetchConfigOptions struct {
	option.Cache
	option.Common
	option.Descriptor
	option.Platform
	option.Pretty
	option.Remote

	outputPath string
	targetRef  string
}

func fetchConfigCmd() *cobra.Command {
	var opts fetchConfigOptions
	cmd := &cobra.Command{
		Use:   "fetch-config [flags] <name>{:<tag>|@<digest>}",
		Short: "[Preview] Fetch the config of a manifest from a remote registry",
		Long: `[Preview] Fetch the config of a manifest from a remote registry

** This command is in preview and under development. **

Example - Fetch the config:
  oras manifest fetch-config localhost:5000/hello:latest

Example - Fetch the config of certain platform:
  oras manifest fetch-config --platform 'linux/arm/v5' localhost:5000/hello:latest

Example - Fetch and print the prettified config:
  oras manifest fetch-config --pretty localhost:5000/hello:latest

Example - Fetch the config and save it to a local file:
  oras manifest fetch-config --output config.json localhost:5000/hello:latest

Example - Fetch the descriptor of the config:
  oras manifest fetch-config --descriptor localhost:5000/hello:latest

Example - Fetch and print the prettified descriptor of the config:
  oras manifest fetch-config --descriptor --pretty localhost:5000/hello:latest
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.outputPath == "-" && opts.OutputDescriptor {
				return errors.New("`--output -` cannot be used with `--descriptor` at the same time")
			}

			return opts.ReadPassword()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return fetchConfig(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.outputPath, "output", "o", "", "write it to standard output")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func fetchConfig(opts fetchConfigOptions) (fetchErr error) {
	ctx, _ := opts.SetLoggerLevel()

	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}

	if repo.Reference.Reference == "" {
		return oerrors.NewErrInvalidReference(repo.Reference)
	}

	targetPlatform, err := opts.Parse()
	if err != nil {
		return err
	}

	src, err := opts.CachedTarget(repo)
	if err != nil {
		return err
	}

	// fetch config descriptor
	configDesc, err := fetchConfigDesc(ctx, src, opts.targetRef, targetPlatform)
	if err != nil {
		return err
	}

	if !opts.OutputDescriptor || opts.outputPath != "" {
		// fetch config content
		contentBytes, err := content.FetchAll(ctx, src, configDesc)
		if err != nil {
			return err
		}

		if opts.outputPath == "" || opts.outputPath == "-" {
			// output config content
			return opts.Output(os.Stdout, contentBytes)
		}

		// save config into the local file if the output path is provided
		if err = os.WriteFile(opts.outputPath, contentBytes, 0666); err != nil {
			return err
		}
	}

	if opts.OutputDescriptor {
		// output config's descriptor
		descBytes, err := json.Marshal(configDesc)
		if err != nil {
			return err
		}
		return opts.Output(os.Stdout, descBytes)
	}

	return nil
}

func fetchConfigDesc(ctx context.Context, src oras.ReadOnlyTarget, reference string, targetPlatform *ocispec.Platform) (ocispec.Descriptor, error) {
	// fetch manifest descriptor and content
	fetchOpts := oras.DefaultFetchBytesOptions
	fetchOpts.TargetPlatform = targetPlatform
	manifestDesc, manifestContent, err := oras.FetchBytes(ctx, src, reference, fetchOpts)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	if !descriptor.IsImageManifest(manifestDesc) {
		return ocispec.Descriptor{}, fmt.Errorf("%q is not an image manifest and does not have a config", manifestDesc.Digest)
	}

	// unmarshal manifest content to extract config descriptor
	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestContent, &manifest); err != nil {
		return ocispec.Descriptor{}, err
	}
	return manifest.Config, nil
}
