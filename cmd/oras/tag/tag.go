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

package tag

import (
	"fmt"

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

type tagOptions struct {
	option.Common
	option.Remote

	srcRef    string
	targetRef string
}

func TagCmd() *cobra.Command {
	var opts tagOptions
	cmd := &cobra.Command{
		Use:   "tag name<:tag|@digest> new_tag",
		Short: "[Preview] tag a manifest in the remote registry",
		Long: `[Preview] tag a manifest in the remote registry
** This command is in preview and under development. **

Example -  Tag a manifest with tag v1.0.1 to repository 'locahost:5000/hello' and tag with 'v1.0.2':
oras manifest push localhost:5000/hello:v1.0.1 v1.0.2
`,
		Args: cobra.ExactArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(_ *cobra.Command, args []string) error {
			opts.srcRef = args[0]
			opts.targetRef = args[1]
			return tagManifest(opts)
		},
	}
	option.ApplyFlags(&opts, cmd.Flags())
	fmt.Println("Tagged", opts.targetRef)
	return cmd
}

func tagManifest(opts tagOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewRepository(opts.srcRef, opts.Common)
	if err != nil {
		return err
	}

	if repo.Reference.Reference == "" {
		return errors.NewErrInvalidReference(repo.Reference)
	}

	oras.Tag(ctx, repo, opts.srcRef, opts.targetRef)

	return nil
}
