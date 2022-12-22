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

	"github.com/opencontainers/go-digest"
	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/option"
)

type showTagsOptions struct {
	option.Remote
	option.Common
	targetRef        string
	last             string
	excludeDigestTag bool
}

func showTagsCmd() *cobra.Command {
	var opts showTagsOptions
	cmd := &cobra.Command{
		Use:   "tags [flags] <name>",
		Short: "[Preview] Show tags of the target repository",
		Long: `[Preview] Show tags of the target repository

** This command is in preview and under development. **

Example - Show tags of the target repository:
  oras repo tags localhost:5000/hello

Example - Show tags in the target repository with digest-like tags hidden:
  oras repo tags --exclude-digest-tag localhost:5000/hello

Example - Show tags of the target repository that include values lexically after last:
  oras repo tags --last "last_tag" localhost:5000/hello
`,
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"show-tags"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return showTags(opts)
		},
	}
	cmd.Flags().StringVar(&opts.last, "last", "", "start after the tag specified by `last`")
	cmd.Flags().BoolVar(&opts.excludeDigestTag, "exclude-digest-tags", false, "exclude all digest-like tags such as 'sha256-aaaa...'")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func showTags(opts showTagsOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}
	if repo.Reference.Reference != "" {
		return fmt.Errorf("unexpected tag or digest %q found in repository reference %q", repo.Reference.Reference, opts.targetRef)
	}
	return repo.Tags(ctx, opts.last, func(tags []string) error {
		for _, tag := range tags {
			if opts.excludeDigestTag && isDigestTag(tag) {
				continue
			}
			fmt.Println(tag)
		}
		return nil
	})
}

func isDigestTag(tag string) bool {
	dgst := strings.Replace(tag, "-", ":", 1)
	_, err := digest.Parse(dgst)
	return err == nil
}
