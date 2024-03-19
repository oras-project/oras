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
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/display/status"
	"oras.land/oras/cmd/oras/internal/display/status/track"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/docker"
	"oras.land/oras/internal/graph"
	"oras.land/oras/internal/registryutil"
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
		Short:   "Copy artifacts from one target to another",
		Long: `Copy artifacts from one target to another

Example - Copy an artifact between registries:
  oras cp localhost:5000/net-monitor:v1 localhost:6000/net-monitor-copy:v1

Example - Download an artifact into an OCI image layout folder:
  oras cp --to-oci-layout localhost:5000/net-monitor:v1 ./downloaded:v1

Example - Upload an artifact from an OCI image layout folder:
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
		Args: oerrors.CheckArgs(argument.Exactly(2), "the source and destination for copying"),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.From.RawReference = args[0]
			refs := strings.Split(args[1], ",")
			opts.To.RawReference = refs[0]
			opts.extraRefs = refs[1:]
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCopy(cmd, &opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.recursive, "recursive", "r", false, "[Preview] recursively copy the artifact and its referrer artifacts")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 3, "concurrency level")
	opts.EnableDistributionSpecFlag()
	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.BinaryTarget)
}

func runCopy(cmd *cobra.Command, opts *copyOptions) error {
	ctx, logger := opts.WithContext(cmd.Context())

	// Prepare source
	src, err := opts.From.NewReadonlyTarget(ctx, opts.Common, logger)
	if err != nil {
		return err
	}
	if err := opts.EnsureSourceTargetReferenceNotEmpty(cmd); err != nil {
		return err
	}

	// Prepare destination
	dst, err := opts.To.NewTarget(opts.Common, logger)
	if err != nil {
		return err
	}
	ctx = registryutil.WithScopeHint(ctx, dst, auth.ActionPull, auth.ActionPush)

	desc, err := doCopy(ctx, src, dst, opts)
	if err != nil {
		return err
	}

	if from, err := digest.Parse(opts.From.Reference); err == nil && from != desc.Digest {
		// correct source digest
		opts.From.RawReference = fmt.Sprintf("%s@%s", opts.From.Path, desc.Digest.String())
	}
	outWriter := cmd.OutOrStdout()
	fmt.Fprintln(outWriter, "Copied", opts.From.AnnotatedReference(), "=>", opts.To.AnnotatedReference())

	if len(opts.extraRefs) != 0 {
		tagNOpts := oras.DefaultTagNOptions
		tagNOpts.Concurrency = opts.concurrency
		if _, err = oras.TagN(ctx, status.NewTagStatusPrinter(dst), opts.To.Reference, opts.extraRefs, tagNOpts); err != nil {
			return err
		}
	}

	fmt.Fprintln(outWriter, "Digest:", desc.Digest)

	return nil
}

func doCopy(ctx context.Context, src oras.ReadOnlyGraphTarget, dst oras.GraphTarget, opts *copyOptions) (ocispec.Descriptor, error) {
	// Prepare copy options
	committed := &sync.Map{}
	extendedCopyOptions := oras.DefaultExtendedCopyOptions
	extendedCopyOptions.Concurrency = opts.concurrency
	extendedCopyOptions.FindPredecessors = func(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		return registry.Referrers(ctx, src, desc, "")
	}

	const (
		promptExists  = "Exists "
		promptCopying = "Copying"
		promptCopied  = "Copied "
		promptSkipped = "Skipped"
		promptMounted = "Mounted"
	)
	srcRepo, srcIsRemote := src.(*remote.Repository)
	dstRepo, dstIsRemote := dst.(*remote.Repository)
	if srcIsRemote && dstIsRemote && srcRepo.Reference.Registry == dstRepo.Reference.Registry {
		extendedCopyOptions.MountFrom = func(ctx context.Context, desc ocispec.Descriptor) ([]string, error) {
			return []string{srcRepo.Reference.Repository}, nil
		}
	}
	if opts.TTY == nil {
		// none TTY output
		extendedCopyOptions.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
			committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
			return status.PrintStatus(desc, promptExists, opts.Verbose)
		}
		extendedCopyOptions.PreCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
			return status.PrintStatus(desc, promptCopying, opts.Verbose)
		}
		extendedCopyOptions.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
			committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
			if err := status.PrintSuccessorStatus(ctx, desc, dst, committed, status.StatusPrinter(promptSkipped, opts.Verbose)); err != nil {
				return err
			}
			return status.PrintStatus(desc, promptCopied, opts.Verbose)
		}
		extendedCopyOptions.OnMounted = func(ctx context.Context, desc ocispec.Descriptor) error {
			committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
			return status.PrintStatus(desc, promptMounted, opts.Verbose)
		}
	} else {
		// TTY output
		tracked, err := track.NewTarget(dst, promptCopying, promptCopied, opts.TTY)
		if err != nil {
			return ocispec.Descriptor{}, err
		}
		defer tracked.Close()
		dst = tracked
		extendedCopyOptions.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
			committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
			return tracked.Prompt(desc, promptExists)
		}
		extendedCopyOptions.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
			committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
			return status.PrintSuccessorStatus(ctx, desc, tracked, committed, func(desc ocispec.Descriptor) error {
				return tracked.Prompt(desc, promptSkipped)
			})
		}
		extendedCopyOptions.OnMounted = func(ctx context.Context, desc ocispec.Descriptor) error {
			committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
			return tracked.Prompt(desc, promptMounted)
		}
	}

	var desc ocispec.Descriptor
	var err error
	rOpts := oras.DefaultResolveOptions
	rOpts.TargetPlatform = opts.Platform.Platform
	if opts.recursive {
		desc, err = oras.Resolve(ctx, src, opts.From.Reference, rOpts)
		if err != nil {
			return ocispec.Descriptor{}, fmt.Errorf("failed to resolve %s: %w", opts.From.Reference, err)
		}
		err = recursiveCopy(ctx, src, dst, opts.To.Reference, desc, extendedCopyOptions)
	} else {
		if opts.To.Reference == "" {
			desc, err = oras.Resolve(ctx, src, opts.From.Reference, rOpts)
			if err != nil {
				return ocispec.Descriptor{}, fmt.Errorf("failed to resolve %s: %w", opts.From.Reference, err)
			}
			err = oras.CopyGraph(ctx, src, dst, desc, extendedCopyOptions.CopyGraphOptions)
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
	return desc, err
}

// recursiveCopy copies an artifact and its referrers from one target to another.
// If the artifact is a manifest list or index, referrers of its manifests are copied as well.
func recursiveCopy(ctx context.Context, src oras.ReadOnlyGraphTarget, dst oras.Target, dstRef string, root ocispec.Descriptor, opts oras.ExtendedCopyOptions) error {
	if root.MediaType == ocispec.MediaTypeImageIndex || root.MediaType == docker.MediaTypeManifestList {
		fetched, err := content.FetchAll(ctx, src, root)
		if err != nil {
			return err
		}
		var index ocispec.Index
		if err = json.Unmarshal(fetched, &index); err != nil {
			return nil
		}

		referrers, err := graph.FindPredecessors(ctx, src, index.Manifests, opts)
		if err != nil {
			return err
		}
		referrers = slices.DeleteFunc(referrers, func(desc ocispec.Descriptor) bool {
			return content.Equal(desc, root)
		})

		findPredecessor := opts.FindPredecessors
		opts.FindPredecessors = func(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
			descs, err := findPredecessor(ctx, src, desc)
			if err != nil {
				return nil, err
			}
			if content.Equal(desc, root) {
				// make sure referrers of child manifests are copied by pointing them to root
				descs = append(descs, referrers...)
			}
			return descs, nil
		}
	}

	var err error
	if dstRef == "" || dstRef == root.Digest.String() {
		err = oras.ExtendedCopyGraph(ctx, src, dst, root, opts.ExtendedCopyGraphOptions)
	} else {
		_, err = oras.ExtendedCopy(ctx, src, root.Digest.String(), dst, dstRef, opts)
	}
	return err
}
