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
	"path"
	"sync"

	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/option"
)

const concurrency = 3

type downloadOptions struct {
	option.Common
	From option.Target
}

func downloadCmd() *cobra.Command {
	var opts downloadOptions
	cmd := &cobra.Command{
		Use:     "download [flags] <source>{:<tag>|@<digest>} [<source>{:<tag>|@<digest>}][...]]",
		Aliases: []string{"download"},
		Short:   "[Preview] Download multiple artifacts",
		Long: `[Preview] Download multiple artifacts

** This command is in preview and under development. **

Example - Downlad artifacts
  oras download localhost:5000/owner/this:v1 localhost:6000/owner/that:v1
`,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.From.RawReference = args[0]
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return download(cmd.Context(), opts, args)
		},
	}
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func download(ctx context.Context, opts downloadOptions, args []string) error {
	ctx, _ = opts.WithContext(ctx)

	var dst oras.GraphTarget

	// Prepare download options
	committed := &sync.Map{}
	extendedCopyOptions := oras.DefaultExtendedCopyOptions
	extendedCopyOptions.Concurrency = concurrency
	extendedCopyOptions.PreCopy = display.StatusPrinter("downloading", opts.Verbose)
	extendedCopyOptions.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		if err := display.PrintSuccessorStatus(ctx, desc, "Skipped", dst, committed, opts.Verbose); err != nil {
			return err
		}
		return display.PrintStatus(desc, "Downloaded ", opts.Verbose)
	}
	extendedCopyOptions.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		return display.PrintStatus(desc, "Exists ", opts.Verbose)
	}

	for _, arg := range args {
		// Prepare source
		from := opts.From
		from.Type = option.TargetTypeRemote
		from.RawReference = arg
		src, err := from.NewReadonlyTarget(ctx, opts.Common)
		if err != nil {
			return err
		}

		if err := from.EnsureReferenceNotEmpty(); err != nil {
			return err
		}

		// Prepare destination
		reference, err := registry.ParseReference(from.RawReference)
		if err != nil {
			return err
		}
		var to option.Target
		to.Type = option.TargetTypeOCILayout
		to.RawReference = path.Join("output", reference.Repository, reference.Reference)
		to.Path = to.RawReference
		to.Reference = reference.Reference
		dst, err = oci.New(to.Path)
		if err != nil {
			return err
		}

		desc, err := src.Resolve(ctx, from.Reference)
		if err != nil {
			return err
		}

		err = oras.CopyGraph(ctx, src, dst, desc, extendedCopyOptions.CopyGraphOptions)
		if err != nil {
			return err
		}

		fmt.Println("Downloaded", from.AnnotatedReference(), "=>", to.AnnotatedReference())
	}

	return nil
}
