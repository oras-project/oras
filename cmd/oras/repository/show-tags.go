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
}

func showTagsCmd() *cobra.Command {
	var opts showTagsOptions
	cmd := &cobra.Command{
		Use:   "show-tags <target-ref>",
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
			opts.targetRef = args[0]
			return showTags(opts)
		},
	}
	// cmd.Flags()
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func showTags(opts showTagsOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}
	repo.Tags(ctx, "", func(tags []string) error {
		for _, tag := range tags {
			fmt.Println(tag)
		}
		return nil
	})
	// list all tags
	return nil
}
