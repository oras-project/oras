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

package blob

import (
	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/option"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blob [command]",
		Short: "[Preview] Blob operations",
	}

	cmd.AddCommand(pushCmd())
	return cmd
}

func pushCmd() *cobra.Command {
	var opts pushBlobOptions
	cmd := &cobra.Command{
		Use:   "push name[@digest] file [flags]",
		Short: "[Preview] Push a blob to remote registry",
		Long: `[Preview] Push a blob to remote registry
** This command is in preview and under development. **

Example - Push blob "hi.txt":
  oras blob push localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5 hi.txt

Example - Push blob to the insecure registry:
  oras blob push localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5 hi.txt --insecure

Example - Push blob to the HTTP registry:
  oras blob push localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5 hi.txt --plain-http		
`,
		Args: cobra.ExactArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			opts.fileRef = args[1]
			return pushBlob(opts)
		},
	}

	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}
