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
	"strings"

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/option"
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
		Short: "[Preview] List the repositories under the registry",
		Long: `[Preview] List the repositories under the registry

** This command is in preview and under development. **

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
			if err := parseRepoPath(&opts, args[0]); err != nil {
				return fmt.Errorf("could not parse repository path: %w", err)
			}
			return listRepository(opts)
		},
	}

	cmd.Flags().StringVar(&opts.last, "last", "", "start after the repository specified by `last`")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func parseRepoPath(opts *repositoryOptions, arg string) error {
	path := strings.TrimSuffix(arg, "/")
	if strings.Contains(path, "/") {
		reference, err := registry.ParseReference(path)
		if err != nil {
			return err
		}
		if reference.Reference != "" {
			return fmt.Errorf("tags or digests should not be provided")
		}
		opts.hostname = reference.Registry
		opts.namespace = reference.Repository + "/"
	} else {
		opts.hostname = path
	}
	return nil
}

func listRepository(opts repositoryOptions) error {
	ctx, _ := opts.SetLoggerLevel()
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
