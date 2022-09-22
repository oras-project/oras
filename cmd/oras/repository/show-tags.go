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

type showTagsOptions struct {
	option.Remote
	option.Common
	targetRef string
	last      string
}

func showTagsCmd() *cobra.Command {
	var opts showTagsOptions
	cmd := &cobra.Command{
		Use:   "show-tags [flags] REPOSITORY",
		Short: "[Preview] Show tags of the target repository",
		Long: `[Preview] Show tags of the target repository

** This command is in preview and under development. **

Example - Show tags of the target repository:
  oras repository show-tags localhost:5000/hello

Example - Show tags of the target repository that include values lexically after last:
  oras repository show-tags --last "last_tag" localhost:5000/hello
`,
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"tags"},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return showTags(opts)
		},
	}
	cmd.Flags().StringVar(&opts.last, "last", "", "start after the tag specified by `last`")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func showTags(opts showTagsOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}
	return repo.Tags(ctx, opts.last, func(tags []string) error {
		for _, tag := range tags {
			fmt.Println(tag)
		}
		return nil
	})
}
