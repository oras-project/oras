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
	rescursive bool

	srcRef string
	dstRef string
}

func copyCmd() *cobra.Command {
	var opts copyOptions
	cmd := &cobra.Command{
		Use:     "copy <from-ref> <to-ref>",
		Aliases: []string{"cp"},
		Short:   "[Preview] Copy artifacts from one target to another",
		Long: `[Preview] Copy artifacts from one target to another

** This command is in preview and under development. **

Examples - Copy the artifact tagged 'v1' from repository 'localhost:5000/net-monitor' to repository 'localhost:5000/net-monitor-copy' 
  oras cp localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1

Examples - Copy the artifact tagged 'v1' and its referrers from repository 'localhost:5000/net-monitor' to 'localhost:5000/net-monitor-copy'
  oras cp -r localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.srcRef = args[0]
			opts.dstRef = args[1]
			return runCopy(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.rescursive, "recursive", "r", false, "recursively copy artifacts and its referrer artifacts")
	opts.src.ApplyFlagsWithPrefix(cmd.Flags(), "from", "source")
	opts.dst.ApplyFlagsWithPrefix(cmd.Flags(), "to", "destination")
	option.ApplyFlags(&opts, cmd.Flags())

	return cmd
}

func runCopy(opts copyOptions) error {
	ctx, _ := opts.SetLoggerLevel()

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
	extendedCopyOptions := oras.DefaultExtendedCopyOptions
	outputStatus := func(status string) func(context.Context, ocispec.Descriptor) error {
		return func(ctx context.Context, desc ocispec.Descriptor) error {
			name, ok := desc.Annotations[ocispec.AnnotationTitle]
			if !ok {
				if !opts.Verbose {
					return nil
				}
				name = desc.MediaType
			}
			return display.Print(status, display.ShortDigest(desc), name)
		}
	}
	extendedCopyOptions.PreCopy = outputStatus("Copying")
	extendedCopyOptions.PostCopy = outputStatus("Copied ")
	extendedCopyOptions.OnCopySkipped = outputStatus("Exists ")

	if src.Reference.Reference == "" {
		return errors.NewErrInvalidReference(src.Reference)
	}

	// push to the destination with digest only if no tag specified
	var desc ocispec.Descriptor
	if ref := dst.Reference.Reference; ref == "" {
		desc, err = src.Resolve(ctx, src.Reference.Reference)
		if err != nil {
			return err
		}
		if opts.rescursive {
			err = oras.ExtendedCopyGraph(ctx, src, dst, desc, extendedCopyOptions.ExtendedCopyGraphOptions)
		} else {
			err = oras.CopyGraph(ctx, src, dst, desc, extendedCopyOptions.CopyGraphOptions)
		}
	} else {
		if opts.rescursive {
			desc, err = oras.ExtendedCopy(ctx, src, opts.srcRef, dst, opts.dstRef, extendedCopyOptions)
		} else {
			copyOptions := oras.CopyOptions{
				CopyGraphOptions: extendedCopyOptions.CopyGraphOptions,
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
