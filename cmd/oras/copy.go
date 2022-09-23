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

package main

import (
	"context"
	"fmt"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

type copyOptions struct {
	src option.Remote
	dst option.Remote
	option.Common
	option.Platform
	recursive bool

	srcRef string
	dstRef string
}

func copyCmd() *cobra.Command {
	var opts copyOptions
	cmd := &cobra.Command{
		Use:     "copy [flags] <from>{:<tag>|@<digest>} <to>[:<tag>|@<digest>]",
		Aliases: []string{"cp"},
		Short:   "[Preview] Copy artifacts from one target to another",
		Long: `[Preview] Copy artifacts from one target to another

** This command is in preview and under development. **

Example - Copy the artifact tagged 'v1' from repository 'localhost:5000/net-monitor' to repository 'localhost:5000/net-monitor-copy' 
  oras copy localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1

Example - Copy the artifact tagged 'v1' and its referrers from repository 'localhost:5000/net-monitor' to 'localhost:5000/net-monitor-copy'
  oras copy -r localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1

Example - Copy the artifact tagged 'v1' from repository 'localhost:5000/net-monitor' to 'localhost:5000/net-monitor-copy' with certain platform
  oras copy --platform linux/arm/v5 localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1 
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.srcRef = args[0]
			opts.dstRef = args[1]
			return runCopy(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.recursive, "recursive", "r", false, "recursively copy artifacts and its referrer artifacts")
	opts.src.ApplyFlagsWithPrefix(cmd.Flags(), "from", "source")
	opts.dst.ApplyFlagsWithPrefix(cmd.Flags(), "to", "destination")
	option.ApplyFlags(&opts, cmd.Flags())

	return cmd
}

func runCopy(opts copyOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	targetPlatform, err := opts.Parse()
	if err != nil {
		return err
	}

	// Prepare source
	src, err := opts.src.NewRepository(opts.srcRef, opts.Common)
	if err != nil {
		return err
	}

	// Prepare destination
	dst, err := opts.dst.NewRepository(opts.dstRef, opts.Common)
	if err != nil {
		return err
	}

	// Prepare copy options
	committed := &sync.Map{}
	extendedCopyOptions := oras.DefaultExtendedCopyOptions
	extendedCopyOptions.PreCopy = display.StatusPrinter("Copying", opts.Verbose)
	extendedCopyOptions.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		if err := display.PrintSuccessorStatus(ctx, desc, "Skipped", dst, committed, opts.Verbose); err != nil {
			return err
		}
		return display.PrintStatus(desc, "Copied ", opts.Verbose)
	}
	extendedCopyOptions.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		return display.PrintStatus(desc, "Exists ", opts.Verbose)
	}

	if src.Reference.Reference == "" {
		return errors.NewErrInvalidReference(src.Reference)
	}

	var desc ocispec.Descriptor
	if ref := dst.Reference.Reference; ref == "" {
		// push to the destination with digest only if no tag specified
		desc, err = src.Resolve(ctx, src.Reference.Reference)
		if err != nil {
			return err
		}
		if opts.recursive {
			err = oras.ExtendedCopyGraph(ctx, src, dst, desc, extendedCopyOptions.ExtendedCopyGraphOptions)
		} else {
			err = oras.CopyGraph(ctx, src, dst, desc, extendedCopyOptions.CopyGraphOptions)
		}
	} else {
		if opts.recursive {
			desc, err = oras.ExtendedCopy(ctx, src, opts.srcRef, dst, opts.dstRef, extendedCopyOptions)
		} else {
			copyOptions := oras.CopyOptions{
				CopyGraphOptions: extendedCopyOptions.CopyGraphOptions,
			}
			if targetPlatform != nil {
				copyOptions.WithTargetPlatform(targetPlatform)
			}
			desc, err = oras.Copy(ctx, src, opts.srcRef, dst, opts.dstRef, copyOptions)
		}
	}
	if err != nil {
		return err
	}

	fmt.Println("Copied", opts.srcRef, "=>", opts.dstRef)
	fmt.Println("Digest:", desc.Digest)

	return nil
}
