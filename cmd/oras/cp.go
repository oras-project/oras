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
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

type copyOptions struct {
	option.Common
	option.Platform
	option.BinaryTarget

	recursive   bool
	concurrency int
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

Example - Copy the artifacts:
  oras cp localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1  # copy between repositories
  oras cp localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy     # copy without tagging in the destination
  oras cp --to-oci localhost:5000/net-monitor:v1 test:v1                    # download into an OCI layout folder 'test'
  oras cp --from-oci test:v1 localhost:5000/net-monitor:v1                  # upload from an OCI layout folder 'test'

Example - Copy the artifact and its referrers:
  oras cp -r localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1  # copy between repositories
  oras cp -r --to-oci localhost:5000/net-monitor:v1 test:v1                    # download into an OCI image layout folder 'test'
  oras cp -r --from-oci test:v1 localhost:5000/net-monitor:v1                  # upload from an OCI image layout folder 'test'

Example - Copy certain platform of an artifact:
  oras cp --platform linux/arm/v5 localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1  # copy between repositories
  oras cp --platform linux/arm/v5 --to-oci localhost:5000/net-monitor:v1 test:v1                    # download into an OCI layout folder 'test'
  oras cp --platform linux/arm/v5 --from-oci test:v1 localhost:5000/net-monitor:v1                  # upload from an OCI layout folder 'test'

Example - Copy the artifact with multiple tags:
  oras cp localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:tag1,tag2,tag3  # copy between repositories
  oras cp localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:tag1,tag2,tag3  # copy between repositories with concurrency level tuned
  oras cp localhost:5000/net-monitor:v1 test:tag1,tag2,tag3 --to-oci                    # download into an OCI layout folder 'test'
  oras cp test:v1 localhost:5000/net-monitor-copy:tag1,tag2,tag3 --from-oci             # upload from an OCI layout folder 'test'
`,
		Args: cobra.ExactArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.From.FQDNReference = args[0]
			refs := strings.Split(args[1], ",")
			opts.To.FQDNReference = refs[0]
			opts.extraRefs = refs[1:]
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
	src, err := opts.From.NewReadonlyTarget(ctx, opts.Common)
	if err != nil {
		return err
	}
	if opts.From.Reference == "" {
		return errors.NewErrInvalidReferenceStr(opts.From.FQDNReference)
	}

	// Prepare destination
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
	if ref := opts.To.Reference; ref == "" {
		// push to the destination with digest only if no tag specified
		desc, err = src.Resolve(ctx, opts.From.Reference)
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
			desc, err = oras.ExtendedCopy(ctx, src, opts.From.Reference, dst, opts.To.Reference, extendedCopyOptions)
		} else {
			copyOptions := oras.CopyOptions{
				CopyGraphOptions: extendedCopyOptions.CopyGraphOptions,
			}
			if opts.Platform.Platform != nil {
				copyOptions.WithTargetPlatform(opts.Platform.Platform)
			}
			desc, err = oras.Copy(ctx, src, opts.From.Reference, dst, opts.To.Reference, copyOptions)
		}
	}
	if err != nil {
		return err
	}

	fmt.Printf("Copied %s => %s \n", opts.From.FullReference(), opts.To.FullReference())

	if len(opts.extraRefs) != 0 {
		tagNOpts := oras.DefaultTagNOptions
		tagNOpts.Concurrency = opts.concurrency
		if _, err = oras.TagN(ctx, &display.TagManifestStatusPrinter{Target: dst}, opts.To.Reference, opts.extraRefs, tagNOpts); err != nil {
			return err
		}
	}

	fmt.Println("Digest:", desc.Digest)

	return nil
}
