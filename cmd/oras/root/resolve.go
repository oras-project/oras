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
	"fmt"

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras/cmd/oras/internal/argument"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

type resolveOptions struct {
	option.Common
	option.Platform
	option.Target

	fullRef bool
}

func resolveCmd() *cobra.Command {
	var opts resolveOptions

	cmd := &cobra.Command{
		Use:   "resolve [flags] <name>{:<tag>|@<digest>}",
		Short: "[Experimental] Resolves digest of the target artifact",
		Long: `[Experimental] Resolves digest of the target artifact

Example - Resolve digest of the target artifact:
  oras resolve localhost:5000/hello-world:v1
`,
		Args:    oerrors.CheckArgs(argument.Exactly(1), "the target artifact reference to resolve"),
		Aliases: []string{"digest"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.RawReference = args[0]
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResolve(cmd, &opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.fullRef, "full-reference", "l", false, "print the full artifact reference with digest")
	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.Target)
}

func runResolve(cmd *cobra.Command, opts *resolveOptions) error {
	ctx, logger := opts.WithContext(cmd.Context())
	repo, err := opts.NewReadonlyTarget(ctx, opts.Common, logger)
	if err != nil {
		return err
	}
	if err := opts.EnsureReferenceNotEmpty(cmd, true); err != nil {
		return err
	}
	resolveOpts := oras.DefaultResolveOptions
	resolveOpts.TargetPlatform = opts.Platform.Platform
	desc, err := oras.Resolve(ctx, repo, opts.Reference, resolveOpts)

	if err != nil {
		return fmt.Errorf("failed to resolve digest: %w", err)
	}

	outWriter := cmd.OutOrStdout()
	if opts.fullRef {
		fmt.Fprintf(outWriter, "%s@%s\n", opts.Path, desc.Digest)
	} else {
		fmt.Fprintln(outWriter, desc.Digest.String())
	}

	return nil
}
