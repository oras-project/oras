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
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/option"
)

type repositoryOptions struct {
	option.Remote
	option.Common
	hostname string
	last     string
	first    int
	skip     int
}

func listCmd() *cobra.Command {
	var opts repositoryOptions
	cmd := &cobra.Command{
		Use:   "list REGISTRY [flags]",
		Short: "[Preview] List the repositories under the registry",
		Long: `[Preview] List the repositories under the registry
** This command is in preview and under development. **
Example - List the repositories under the registry:
  oras repository list localhost:5000
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.hostname = args[0]
			return listRepository(opts)
		},
	}

	option.ApplyFlags(&opts, cmd.Flags())
	cmd.Flags().StringVar(&opts.last, "last", "", "start after the repository specified by `last`")
	cmd.Flags().IntVar(&opts.first, "first", 1000, "the first X records")
	cmd.Flags().IntVar(&opts.skip, "skip", 0, "skip the first X record")
	return cmd
}

func listRepository(opts repositoryOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	reg, err := opts.Remote.NewRegistry(opts.hostname, opts.Common)
	if err != nil {
		return err
	}
	if err := reg.Repositories(ctx, opts.last, func(repos []string) error {
		repos = display.Cut(repos, opts.first, opts.skip)
		for _, repo := range repos {
			fmt.Println(repo)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
