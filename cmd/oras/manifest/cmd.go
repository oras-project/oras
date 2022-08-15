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
	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/option"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manifest [fetch]",
		Short: "[Preview] Manifest operations",
	}

	cmd.AddCommand(
		fetchCmd(),
	)
	return cmd
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
		RunE: func(_ *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return fetchManifest(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.pretty, "pretty", "", false, "output prettified manifest")
	cmd.Flags().BoolVarP(&opts.fetchDescriptor, "descriptor", "", false, "fetch a descriptor of the manifest")
	cmd.Flags().IntVarP(&opts.indent, "indent", "n", 2, "number of spaces for indentation")
	cmd.Flags().StringVarP(&opts.mediaType, "media-type", "", "", "accepted media types")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}
