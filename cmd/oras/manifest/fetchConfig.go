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
	"fmt"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras/cmd/oras/internal/descriptor"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/cache"
)

type fetchConfigOptions struct {
	option.Common
	option.Descriptor
	option.Pretty
	option.Remote

	cacheRoot  string
	mediaType  string
	outputPath string
	targetRef  string
}

func fetchConfigCmd() *cobra.Command {
	var opts fetchConfigOptions
	cmd := &cobra.Command{
		Use:   "fetch-config name{:tag|@digest}",
		Short: "[Preview] Fetch the config of a manifest from a remote registry",
		Long: `[Preview] Fetch the config of a manifest from a remote registry

** This command is in preview and under development. **

Example - Fetch the config:
  oras manifest fetch-config localhost:5000/hello:latest

Example - Fetch the config and save it to a local file:
  oras manifest fetch-config localhost:5000/hello:latest --output config.json

Example - Fetch the descriptor of the config:
oras manifest fetch-config localhost:5000/hello:latest --descriptor
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.cacheRoot = os.Getenv("ORAS_CACHE")
			return opts.ReadPassword()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return fetchConfig(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.outputPath, "output", "o", "", "output file path")
	cmd.Flags().StringVarP(&opts.mediaType, "media-type", "", "", "media type of the manifest config")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func fetchConfig(opts fetchConfigOptions) (err error) {
	ctx, _ := opts.SetLoggerLevel()

	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}

	if repo.Reference.Reference == "" {
		return errors.NewErrInvalidReference(repo.Reference)
	}

	var src oras.ReadOnlyTarget = repo
	if opts.cacheRoot != "" {
		ociStore, err := oci.New(opts.cacheRoot)
		if err != nil {
			return err
		}
		src = cache.New(repo, ociStore)
	}

	configDesc, err := fetchConfigDesc(ctx, src, opts.targetRef, opts.mediaType)
	if err != nil {
		return err
	}

	// outputs config's descriptor if `--descriptor` is used
	if opts.OutputDescriptor {
		descBytes, err := json.Marshal(configDesc)
		if err != nil {
			return err
		}
		err = opts.Output(os.Stdout, descBytes)
		if err != nil {
			return err
		}
	}

	contentBytes, err := content.FetchAll(ctx, src, configDesc)
	if err != nil {
		return err
	}

	// save config into the local file if the output path is provided
	if opts.outputPath != "" {
		file, err := os.OpenFile(opts.outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return err
		}
		defer func() {
			if closeErr := file.Close(); err == nil {
				err = closeErr
			}
		}()

		_, err = file.Write(contentBytes)
		if err != nil {
			return err
		}
	}

	if !opts.OutputDescriptor && opts.outputPath == "" {
		err = opts.Output(os.Stdout, contentBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

func fetchConfigDesc(ctx context.Context, src oras.ReadOnlyTarget, reference string, configMediaType string) (ocispec.Descriptor, error) {
	// fetch manifest descriptor
	manifestDesc, err := oras.Resolve(ctx, src, reference, oras.ResolveOptions{})
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	// fetch config descriptor
	successors, err := content.Successors(ctx, src, manifestDesc)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	for i, s := range successors {
		if s.MediaType == configMediaType || (configMediaType == "" && i == 0 && descriptor.IsImageManifest(manifestDesc.MediaType)) {
			return s, nil
		}
	}
	return ocispec.Descriptor{}, fmt.Errorf("%s does not have a config", reference)
}
