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

package repository

import (
	"fmt"

	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/option"
)

type repositoryOptions struct {
	option.Remote
	option.Common
	hostname string
}

func listCmd() *cobra.Command {
	var opts repositoryOptions
	// in case need option
	cmd := &cobra.Command{
		Use:   "list [REGISTRY]",
		Short: "[Preview] List the repositories under the registry",
		Long: `[Preview] List the repositories under the registry
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
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.hostname = args[0]
			return listRepository(opts)
		},
	}

	// cmd.Flags()
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func listRepository(opts repositoryOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	// get all repository from the registry
	reg, err := opts.Remote.NewRegistry(opts.hostname, opts.Common)
	// https://docs.docker.com/registry/spec/api/#catalog
	if err != nil {
		return err
	}
	// RepositoryListPageSize
	reg.Repositories(ctx, "", func(repos []string) error {
		for _, repo := range repos {
			fmt.Println(repo)
		}
		return nil
	})
	// list all repository
	return nil
}
