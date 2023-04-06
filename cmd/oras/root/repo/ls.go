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

package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/repository"
)

type repositoryOptions struct {
	option.Remote
	option.Common
	hostname  string
	namespace string
	last      string
}

func listCmd() *cobra.Command {
	var opts repositoryOptions
	cmd := &cobra.Command{
		Use:   "ls [flags] <registry>",
		Short: "List the repositories under the registry",
		Long: `List the repositories under the registry

Example - List the repositories under the registry:
  oras repo ls localhost:5000

Example - List the repositories under a namespace in the registry:
  oras repo ls localhost:5000/example-namespace

Example - List the repositories under the registry that include values lexically after last:
  oras repo ls --last "last_repo" localhost:5000
`,
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"list"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if opts.hostname, opts.namespace, err = repository.ParseRepoPath(args[0]); err != nil {
				return fmt.Errorf("could not parse repository path: %w", err)
			}
			return listRepository(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.last, "last", "", "start after the repository specified by `last`")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func listRepository(ctx context.Context, opts repositoryOptions) error {
	ctx, _ = opts.WithContext(ctx)
	reg, err := opts.Remote.NewRegistry(opts.hostname, opts.Common)
	if err != nil {
		return err
	}
	return reg.Repositories(ctx, opts.last, func(repos []string) error {
		for _, repo := range repos {
			if subRepo, found := strings.CutPrefix(repo, opts.namespace); found {
				fmt.Println(subRepo)
			}
		}
		return nil
	})
}
