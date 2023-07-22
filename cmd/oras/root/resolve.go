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
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras/cmd/oras/internal/option"
)

type resolveOptions struct {
	option.Common
	option.Platform
	option.Target

	FullRef bool
}

func resolveCmd() *cobra.Command {
	var opts resolveOptions

	cmd := &cobra.Command{
		Use:     "resolve [flags] <name>:<tag>",
		Short:   "Resolves digest of the target artifact",
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"digest"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.RawReference = args[0]
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResolve(cmd.Context(), opts)
		},
	}

	cmd.Flags().BoolVar(&opts.FullRef, "full-ref", false, "print the full artifact reference with digest")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runResolve(ctx context.Context, opts resolveOptions) error {
	ctx, _ = opts.WithContext(ctx)
	repo, err := opts.NewReadonlyTarget(ctx, opts.Common)
	if err != nil {
		return err
	}
	if err := opts.EnsureReferenceNotEmpty(); err != nil {
		return err
	}
	resolveOpts := oras.DefaultResolveOptions
	resolveOpts.TargetPlatform = opts.Platform.Platform
	desc, err := oras.Resolve(ctx, repo, opts.Reference, resolveOpts)

	if err != nil {
		return fmt.Errorf("failed to resolve the digest: %w", err)
	}

	if opts.FullRef {
		fmt.Printf("%s@%s\n", opts.RawReference, desc.Digest.String())
	} else {
		fmt.Println(desc.Digest.String())
	}

	return nil
}