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

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/display"
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

Example - Copy an artifact between registries:
  oras cp localhost:5000/net-monitor:v1 localhost:6000/net-monitor-copy:v1

Example - Download an artifact into an OCI layout folder:
  oras cp --to-oci-layout localhost:5000/net-monitor:v1 ./downloaded:v1

Example - Upload an artifact from an OCI layout folder:
  oras cp --from-oci-layout ./to-upload:v1 localhost:5000/net-monitor:v1

Example - Upload an artifact from an OCI layout tar archive:
  oras cp --from-oci-layout ./to-upload.tar:v1 localhost:5000/net-monitor:v1

Example - Copy an artifact and its referrers:
  oras cp -r localhost:5000/net-monitor:v1 localhost:6000/net-monitor-copy:v1

Example - Copy an artifact and referrers using specific methods for the Referrers API:
  oras cp -r --from-distribution-spec v1.1-referrers-api --to-distribution-spec v1.1-referrers-tag \
    localhost:5000/net-monitor:v1 localhost:6000/net-monitor-copy:v1 

Example - Copy certain platform of an artifact:
  oras cp --platform linux/arm/v5 localhost:5000/net-monitor:v1 localhost:6000/net-monitor-copy:v1

Example - Copy an artifact with multiple tags:
  oras cp localhost:5000/net-monitor:v1 localhost:6000/net-monitor-copy:tag1,tag2,tag3

Example - Copy an artifact with multiple tags with concurrency tuned:
  oras cp --concurrency 10 localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:tag1,tag2,tag3
`,
		Args: cobra.ExactArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.From.RawReference = args[0]
			refs := strings.Split(args[1], ",")
			opts.To.RawReference = refs[0]
			opts.extraRefs = refs[1:]
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCopy(opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.recursive, "recursive", "r", false, "recursively copy the artifact and its referrer artifacts")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 3, "concurrency level")
	opts.EnableDistributionSpecFlag()
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
	if err := opts.From.EnsureReferenceNotEmpty(); err != nil {
		return err
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
	extendedCopyOptions.FindPredecessors = func(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		var results []ocispec.Descriptor
		predecessors, err := src.Predecessors(ctx, desc)
		if err != nil {
			return nil, err
		}
		for _, node := range predecessors {
			switch node.MediaType {
			case ocispec.MediaTypeArtifactManifest, ocispec.MediaTypeImageManifest:
				results = append(results, node)
			}
		}
		return results, nil
	}
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
	if ref := opts.To.Reference; ref == "" || (opts.recursive && opts.Platform.Platform != nil) {
		// need resolving first
		rOpts := oras.DefaultResolveOptions
		rOpts.TargetPlatform = opts.Platform.Platform
		desc, err = oras.Resolve(ctx, src, opts.From.Reference, rOpts)
		if err != nil {
			return err
		}
		if opts.recursive {
			// not ExtendCopy since we already have desc
			err = oras.ExtendedCopyGraph(ctx, src, dst, desc, extendedCopyOptions.ExtendedCopyGraphOptions)
		} else {
			err = oras.CopyGraph(ctx, src, dst, desc, extendedCopyOptions.CopyGraphOptions)
		}
		if err != nil {
			return err
		}
		if ref != "" {
			err = dst.Tag(ctx, desc, ref)
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
			desc, err = oras.Copy(ctx, src, opts.From.Reference, dst, ref, copyOptions)
		}
	}
	if err != nil {
		return err
	}

	if from, err := digest.Parse(opts.From.Reference); err == nil && from != desc.Digest {
		// correct source digest
		opts.From.RawReference = fmt.Sprintf("%s@%s", opts.From.Path, desc.Digest.String())
	}
	fmt.Println("Copied", opts.From.AnnotatedReference(), "=>", opts.To.AnnotatedReference())

	if len(opts.extraRefs) != 0 {
		tagNOpts := oras.DefaultTagNOptions
		tagNOpts.Concurrency = opts.concurrency
		if _, err = oras.TagN(ctx, display.NewTagManifestStatusPrinter(dst), opts.To.Reference, opts.extraRefs, tagNOpts); err != nil {
			return err
		}
	}

	fmt.Println("Digest:", desc.Digest)

	return nil
}
