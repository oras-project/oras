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
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/cas"
)

type fetchOptions struct {
	option.Common
	option.Remote
	option.Platform

	targetRef       string
	pretty          bool
	mediaTypes      []string
	fetchDescriptor bool
}

func fetchCmd() *cobra.Command {
	var opts fetchOptions
	cmd := &cobra.Command{
		Use:   "fetch [flags] <name:tag|name@digest>",
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
			return opts.ReadPassword()
		},
		Aliases: []string{"get"},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return fetchManifest(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.pretty, "pretty", "", false, "output prettified manifest")
	cmd.Flags().BoolVarP(&opts.fetchDescriptor, "descriptor", "", false, "fetch a descriptor of the manifest")
	cmd.Flags().StringSliceVarP(&opts.mediaTypes, "media-type", "", nil, "accepted media types")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func fetchManifest(opts fetchOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	tagetPlatform, err := opts.Parse()
	if err != nil {
		return err
	}
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}
	if repo.Reference.Reference == "" {
		return errors.NewErrInvalidReference(repo.Reference)
	}
	repo.ManifestMediaTypes = opts.mediaTypes

	// Fetch and output
	var content []byte
	if opts.fetchDescriptor {
		content, err = cas.FetchDescriptor(ctx, repo, opts.targetRef, tagetPlatform)
	} else {
		content, err = cas.FetchManifest(ctx, repo, opts.targetRef, tagetPlatform)
	}
	if err != nil {
		return err
	}
	if opts.pretty {
		buf := bytes.NewBuffer(nil)
		if err = json.Indent(buf, content, "", "  "); err != nil {
			return fmt.Errorf("failed to prettify: %w", err)
		}
		content = buf.Bytes()
	}
	_, err = os.Stdout.Write(content)
	return err
}
