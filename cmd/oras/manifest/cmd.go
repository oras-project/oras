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
		Use:   "manifest [command]",
		Short: "[Preview] Manifest operations",
	}

	cmd.AddCommand(pushCmd())
	return cmd
}

func pushCmd() *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:   "push name[:tag|@digest] file",
		Short: "[Preview] Push a manifest to remote registry",
		Long: `[Preview] Push a manifest to remote registry
** This command is in preview and under development. **

Example - Push manifest:
  oras manifest push localhost:5000/hello:latest manifest.json
`,
		Args: cobra.ExactArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(_ *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			opts.fileRef = args[1]
			return pushManifest(opts)
		},
	}

	option.ApplyFlags(&opts, cmd.Flags())
	cmd.Flags().StringVarP(&opts.mediaType, "media-type", "", "", "media type of manifest")
	return cmd
}
