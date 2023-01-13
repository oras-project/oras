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

Example - Copy the artifact tagged with 'v1' from repository 'localhost:5000/net-monitor' to repository 'localhost:5000/net-monitor-copy':
  oras cp localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1

Example - Copy the artifact tagged with 'v1' and its referrers from repository 'localhost:5000/net-monitor' to 'localhost:5000/net-monitor-copy':
  oras cp -r localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1

Example - Copy the artifact tagged with 'v1' from repository 'localhost:5000/net-monitor' to 'localhost:5000/net-monitor-copy' with certain platform:
  oras cp --platform linux/arm/v5 localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1 

Example - Copy the artifact tagged with 'v1' from repository 'localhost:5000/net-monitor' to 'localhost:5000/net-monitor-copy' with multiple tags:
  oras cp localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1,tag2,tag3

Example - Copy the artifact tagged with 'v1' from repository 'localhost:5000/net-monitor' to 'localhost:5000/net-monitor-copy' with multiple tags and concurrency level tuned:
  oras cp --concurrency 6 localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1,tag2,tag3

Example - Download an artifact from remote registry to a folder 'local' in OCI image layout:
  oras cp --to-oci localhost:5000/net-monitor:v1 local:v1
  oras cp --to-target type=oci localhost:5000/net-monitor:v1 local:v1

Example - Download an artifact and its referrers from remote registry to a folder 'local' in OCI image layout:
  oras cp --to-oci -r localhost:5000/net-monitor:v1 local:v1

Example - Download certain platform of an artifact from remote registry to a folder 'local' in OCI image layout:
  oras cp --to-oci --platform linux/arm/v5 localhost:5000/net-monitor:v1 local:v1 

Example - Download an artifact from remote registry to a folder 'local' in OCI image layout with multiple tags:
  oras cp --to-oci localhost:5000/net-monitor:v1 local:tag1,tag2,tag3

Example - Download an artifact from remote registry to a folder 'local' in OCI image layout with multiple tags and concurrency level tuned:
  oras cp --to-oci --concurrency 6 localhost:5000/net-monitor:v1 local:tag1,tag2,tag3

Example - Upload an artifact a folder 'local' in OCI image layout to remote registry:
  oras cp --from-oci local:v1  localhost:5000/net-monitor:v1
  oras cp --from-target type=oci local:v1  localhost:5000/net-monitor:v1

Example - Upload an artifact a tar archive in OCI image layout to remote registry:
  oras cp --from-oci local.tar  localhost:5000/net-monitor:v1

Example - Upload an artifact and its referrers from a folder 'local' in OCI image layout to remote registry:
  oras cp --from-oci -r local:v1  localhost:5000/net-monitor:v1

Example - Upload certain platform of an artifact from a folder 'local' in OCI image layout to remote registry:
  oras cp --from-oci --platform linux/arm/v5 local:v1  localhost:5000/net-monitor:v1

Example - Upload an artifact from a folder 'local' in OCI image layout to remote registry with multiple tags:
  oras cp --from-oci local:v1 localhost:5000/net-monitor:tag1,tag2,tag3

Example - Upload an artifact from a folder 'local' in OCI image layout to remote registry with multiple tags and concurrency level tuned:
  oras cp --concurrency 6 --from-oci local:v1 localhost:5000/net-monitor:tag1,tag2,tag3
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
