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

package root

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/display/status"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

type tagOptions struct {
	option.Common
	option.Target

	concurrency int
	targetRefs  []string
}

func tagCmd() *cobra.Command {
	var opts tagOptions
	cmd := &cobra.Command{
		Use:   "tag [flags] <name>{:<tag>|@<digest>} <new_tag> [...]",
		Short: "Tag a manifest in a registry or an OCI image layout",
		Long: `Tag a manifest in a registry or an OCI image layout

Example - Tag the manifest 'v1.0.1' in 'localhost:5000/hello' to 'v1.0.2':
  oras tag localhost:5000/hello:v1.0.1 v1.0.2

Example - Tag the manifest with digest sha256:9463e0d192846bc994279417b50114606712d516aab45f4d8b31cbc6e46aad71 to 'v1.0.2'
  oras tag localhost:5000/hello@sha256:9463e0d192846bc994279417b50114606712d516aab45f4d8b31cbc6e46aad71 v1.0.2

Example - Tag the manifest 'v1.0.1' in 'localhost:5000/hello' to 'v1.0.2', 'latest'
  oras tag localhost:5000/hello:v1.0.1 v1.0.2 latest

Example - Tag the manifest 'v1.0.1' in 'localhost:5000/hello' to 'v1.0.1', 'v1.0.2', 'latest' with concurrency level tuned:
  oras tag --concurrency 1 localhost:5000/hello:v1.0.1 v1.0.2 latest

Example - Tag the manifest 'v1.0.1' to 'v1.0.2' in an OCI image layout folder 'layout-dir':
  oras tag layout-dir:v1.0.1 v1.0.2
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && (args[0] == "list" || args[0] == "ls") {
				container := "a repository"
				if opts.IsOCILayout {
					container = "an OCI image layout"
				}
				return &oerrors.Error{
					Err:            errors.New(`there is no "list" sub-command for "oras tag" command`),
					Usage:          fmt.Sprintf("%s %s", cmd.Parent().CommandPath(), cmd.Use),
					Recommendation: fmt.Sprintf(`If you want to list available tags in %s, use "oras repo tags"`, container),
				}
			}
			return oerrors.CheckArgs(argument.AtLeast(1), "the artifact to be retagged and the tags to be added")(cmd, args)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.RawReference = args[0]
			opts.targetRefs = args[1:]
			if err := option.Parse(&opts); err != nil {
				if inner, ok := err.(*oerrors.Error); ok {
					if errors.Is(inner, errdef.ErrInvalidReference) {
						inner.Err = fmt.Errorf("unable to add tag for '%s': %w", opts.RawReference, inner.Err)
					}
				}
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return tagManifest(cmd, &opts)
		},
	}

	option.ApplyFlags(&opts, cmd.Flags())
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 5, "concurrency level")
	return oerrors.Command(cmd, &opts.Target)
}

func tagManifest(cmd *cobra.Command, opts *tagOptions) error {
	ctx, logger := opts.WithContext(cmd.Context())
	target, err := opts.NewTarget(opts.Common, logger)
	if err != nil {
		return err
	}
	if err := opts.EnsureReferenceNotEmpty(cmd, true); err != nil {
		return err
	}

	tagNOpts := oras.DefaultTagNOptions
	tagNOpts.Concurrency = opts.concurrency
	_, err = oras.TagN(
		ctx,
		status.NewTagStatusHintPrinter(target, fmt.Sprintf("[%s] %s", opts.Type, opts.Path)),
		opts.Reference,
		opts.targetRefs,
		tagNOpts,
	)
	return err
}
