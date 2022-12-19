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
	"strings"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

type copyOptions struct {
	option.BinaryTarget
	option.Common
	option.Platform
	recursive bool

	concurrency int
	srcRef      string
	dstRef      string
	extraRefs   []string
}

func copyCmd() *cobra.Command {
	var opts copyOptions
	cmd := &cobra.Command{
		Use:     "cp [flags] <from>{:<tag>|@<digest>} <to>[:<tag>[,<tag>][...]]",
		Aliases: []string{"copy"},
		Short:   "[Preview] Copy artifacts from one target to another",
		Long: `[Preview] Copy artifacts from one target to another

** This command is in preview and under development. **

Example - Copy the artifact tagged with 'v1' from repository 'localhost:5000/net-monitor' to repository 'localhost:5000/net-monitor-copy' 
  oras cp localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1

Example - Copy the artifact tagged with 'v1' and its referrers from repository 'localhost:5000/net-monitor' to 'localhost:5000/net-monitor-copy'
  oras cp -r localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1

Example - Copy the artifact tagged with 'v1' from repository 'localhost:5000/net-monitor' to 'localhost:5000/net-monitor-copy' with certain platform
  oras cp --platform linux/arm/v5 localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1 

Example - Copy the artifact tagged with 'v1' from repository 'localhost:5000/net-monitor' to 'localhost:5000/net-monitor-copy' with multiple tags
  oras cp localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1,tag2,tag3

Example - Copy the artifact tagged with 'v1' from repository 'localhost:5000/net-monitor' to 'localhost:5000/net-monitor-copy' with multiple tags and concurrency level tuned
  oras cp --concurrency 6 localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1,tag2,tag3
`,
		Args: cobra.ExactArgs(2),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			opts.srcRef = args[0]
			refs := strings.Split(args[1], ",")
			opts.dstRef = refs[0]
			opts.extraRefs = refs[1:]
			opts.BinaryTarget.SetReferenceInput(opts.srcRef, opts.dstRef)
			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCopy(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.recursive, "recursive", "r", false, "recursively copy the artifact and its referrer artifacts")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 3, "concurrency level")
	option.ApplyFlags(&opts, cmd.Flags())

	return cmd
}

func runCopy(opts copyOptions) error {
	ctx, _ := opts.SetLoggerLevel()

	// Prepare source
	srcRef, err := registry.ParseReference(opts.srcRef)
	if err != nil {
		return err
	}
	src, err := opts.From.NewReadonlyTarget(ctx, opts.Common)
	if err != nil {
		return err
	}
	if opts.From.Reference == "" {
		return errors.NewErrInvalidReferenceStr(opts.From.Fqdn)
	}

	// Prepare destination
	dstRef, err := registry.ParseReference(opts.dstRef)
	if err != nil {
		return err
	}
	dst, err := opts.To.NewTarget(opts.Common)
	if err != nil {
		return err
	}

	// Prepare copy options
	committed := &sync.Map{}
	extendedCopyOptions := oras.DefaultExtendedCopyOptions
	extendedCopyOptions.Concurrency = opts.concurrency
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

	var desc ocispec.Descriptor
	if ref := dstRef.Reference; ref == "" {
		// push to the destination with digest only if no tag specified
		desc, err = src.Resolve(ctx, srcRef.Reference)
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
			copyOptions.WithTargetPlatform(opts.OCIPlatform)
			desc, err = oras.Copy(ctx, src, opts.srcRef, dst, opts.dstRef, copyOptions)
		}
	}
	if err != nil {
		return err
	}

	fmt.Println("Copied", opts.srcRef, "=>", opts.dstRef)

	if len(opts.extraRefs) != 0 {
		tagNOpts := oras.DefaultTagNOptions
		tagNOpts.Concurrency = opts.concurrency
		if err = oras.TagN(ctx, &display.TagManifestStatusPrinter{GraphTarget: dst}, opts.dstRef, opts.extraRefs, tagNOpts); err != nil {
			return err
		}
	}

	fmt.Println("Digest:", desc.Digest)

	return nil
}
