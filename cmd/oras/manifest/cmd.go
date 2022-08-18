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

	cmd.AddCommand(deleteCmd())
	return cmd
}

func deleteCmd() *cobra.Command {
	var opts deleteOptions
	cmd := &cobra.Command{
		Use:   "delete name[:tag|@digest]",
		Short: "[Preview] Delete a manifest from remote registry",
		Long: `[Preview] Delete a manifest from remote registry
** This command is in preview and under development. **

Example - Delete manifest:
  oras manifest delete localhost:5000/hello:latest
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(_ *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return deleteManifest(opts)
		},
	}

	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}
