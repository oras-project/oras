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

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/command"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/display/status"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/docker"
	"oras.land/oras/internal/graph"
	"oras.land/oras/internal/listener"
	"oras.land/oras/internal/registryutil"
)

type copyOptions struct {
	option.Common
	option.Platform
	option.BinaryTarget
	option.Terminal

	recursive   bool
	concurrency int
	extraRefs   []string
	// Deprecated: verbose is deprecated and will be removed in the future.
	verbose bool
}

func copyCmd() *cobra.Command {
	var opts copyOptions
	cmd := &cobra.Command{
		Use:     "cp [flags] <from>{:<tag>|@<digest>} <to>[:<tag>[,<tag>][...]]",
		Aliases: []string{"copy"},
		Short:   "Copy artifacts from one target to another",
		Long: `Copy artifacts from one target to another. When copying an image index, all of its manifests will be copied

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
			err := option.Parse(cmd, &opts)
			if err != nil {
				return err
			}
			opts.DisableTTY(opts.Debug, false)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Printer.Verbose = opts.verbose
			return runCopy(cmd, &opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.recursive, "recursive", "r", false, "[Preview] recursively copy the artifact and its referrer artifacts")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 3, "concurrency level")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", true, "print status output for unnamed blobs")
	_ = cmd.Flags().MarkDeprecated("verbose", "and will be removed in a future release.")
	opts.EnableDistributionSpecFlag()
	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.BinaryTarget)
}

func runCopy(cmd *cobra.Command, opts *copyOptions) error {
	ctx, logger := command.GetLogger(cmd, &opts.Common)

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
	statusHandler, metadataHandler := display.NewCopyHandler(opts.Printer, opts.TTY, dst)

	desc, err := doCopy(ctx, statusHandler, src, dst, opts)
	if err != nil {
		return err
	}

	if from, err := digest.Parse(opts.From.Reference); err == nil && from != desc.Digest {
		// correct source digest
		opts.From.RawReference = fmt.Sprintf("%s@%s", opts.From.Path, desc.Digest.String())
	}

	if err := metadataHandler.OnCopied(&opts.BinaryTarget, desc); err != nil {
		return err
	}

	if len(opts.extraRefs) != 0 {
		tagNOpts := oras.DefaultTagNOptions
		tagNOpts.Concurrency = opts.concurrency
		tagListener := listener.NewTaggedListener(dst, metadataHandler.OnTagged)
		if _, err = oras.TagN(ctx, tagListener, opts.To.Reference, opts.extraRefs, tagNOpts); err != nil {
			return err
		}
	}

	return metadataHandler.Render()
}

func doCopy(ctx context.Context, copyHandler status.CopyHandler, src oras.ReadOnlyGraphTarget, dst oras.GraphTarget, opts *copyOptions) (desc ocispec.Descriptor, err error) {
	// Prepare copy options
	extendedCopyGraphOptions := oras.DefaultExtendedCopyGraphOptions
	extendedCopyGraphOptions.Concurrency = opts.concurrency
	extendedCopyGraphOptions.FindPredecessors = func(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		return registry.Referrers(ctx, src, desc, "")
	}

	srcRepo, srcIsRemote := src.(*remote.Repository)
	dstRepo, dstIsRemote := dst.(*remote.Repository)
	if srcIsRemote && dstIsRemote && srcRepo.Reference.Registry == dstRepo.Reference.Registry {
		extendedCopyGraphOptions.MountFrom = func(ctx context.Context, desc ocispec.Descriptor) ([]string, error) {
			return []string{srcRepo.Reference.Repository}, nil
		}
	}
	dst, err = copyHandler.StartTracking(dst)
	if err != nil {
		return desc, err
	}
	defer func() {
		stopErr := copyHandler.StopTracking()
		if err == nil {
			err = stopErr
		}
	}()
	extendedCopyGraphOptions.OnCopySkipped = copyHandler.OnCopySkipped
	extendedCopyGraphOptions.PreCopy = copyHandler.PreCopy
	extendedCopyGraphOptions.PostCopy = copyHandler.PostCopy
	extendedCopyGraphOptions.OnMounted = copyHandler.OnMounted

	rOpts := oras.DefaultResolveOptions
	rOpts.TargetPlatform = opts.Platform.Platform
	if opts.recursive {
		desc, err = oras.Resolve(ctx, src, opts.From.Reference, rOpts)
		if err != nil {
			return ocispec.Descriptor{}, fmt.Errorf("failed to resolve %s: %w", opts.From.Reference, err)
		}
		err = recursiveCopy(ctx, src, dst, opts.To.Reference, desc, extendedCopyGraphOptions)
	} else {
		if opts.To.Reference == "" {
			desc, err = oras.Resolve(ctx, src, opts.From.Reference, rOpts)
			if err != nil {
				return ocispec.Descriptor{}, fmt.Errorf("failed to resolve %s: %w", opts.From.Reference, err)
			}
			err = oras.CopyGraph(ctx, src, dst, desc, extendedCopyGraphOptions.CopyGraphOptions)
		} else {
			copyOptions := oras.CopyOptions{
				CopyGraphOptions: extendedCopyGraphOptions.CopyGraphOptions,
			}
			if opts.Platform.Platform != nil {
				copyOptions.WithTargetPlatform(opts.Platform.Platform)
			}
			desc, err = oras.Copy(ctx, src, opts.From.Reference, dst, opts.To.Reference, copyOptions)
		}
	}
	// leave the CopyError to oerrors.Modifier for prefix processing
	return desc, err
}

// recursiveCopy copies an artifact and its referrers from one target to another.
// If the artifact is a manifest list or index, referrers of its manifests are copied as well.
func recursiveCopy(ctx context.Context, src oras.ReadOnlyGraphTarget, dst oras.Target, dstRef string, root ocispec.Descriptor, opts oras.ExtendedCopyGraphOptions) error {
	opts, copyRoot, err := prepareCopyOption(ctx, src, dst, root, opts)
	if err != nil {
		return err
	}
	if err := oras.ExtendedCopyGraph(ctx, src, dst, copyRoot, opts); err != nil {
		return err
	}
	if dstRef != "" && dstRef != root.Digest.String() {
		return dst.Tag(ctx, root, dstRef)
	}
	return nil
}

func prepareCopyOption(ctx context.Context, src oras.ReadOnlyGraphTarget, dst oras.Target, root ocispec.Descriptor, opts oras.ExtendedCopyGraphOptions) (oras.ExtendedCopyGraphOptions, ocispec.Descriptor, error) {
	if root.MediaType != ocispec.MediaTypeImageIndex && root.MediaType != docker.MediaTypeManifestList {
		return opts, root, nil
	}

	fetched, err := content.FetchAll(ctx, src, root)
	if err != nil {
		return oras.ExtendedCopyGraphOptions{}, ocispec.Descriptor{}, err
	}
	var index ocispec.Index
	if err = json.Unmarshal(fetched, &index); err != nil {
		return oras.ExtendedCopyGraphOptions{}, ocispec.Descriptor{}, err
	}

	if len(index.Manifests) == 0 {
		// no child manifests, thus no child referrers
		return opts, root, nil
	}

	referrers, err := graph.FindPredecessors(ctx, src, index.Manifests, opts)
	if err != nil {
		return oras.ExtendedCopyGraphOptions{}, ocispec.Descriptor{}, err
	}

	referrers = slices.DeleteFunc(referrers, func(desc ocispec.Descriptor) bool {
		return content.Equal(desc, root)
	})

	if len(referrers) == 0 {
		// no child referrers
		return opts, root, nil
	}

	if opts.FindPredecessors == nil {
		opts.FindPredecessors = func(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
			return registry.Referrers(ctx, src, desc, "")
		}
	}
	rootReferrers, err := opts.FindPredecessors(ctx, src, root)
	if err != nil {
		return oras.ExtendedCopyGraphOptions{}, ocispec.Descriptor{}, err
	}

	// If root has no referrers, we set copyRoot, which is the entry point of
	// extended copy, to the first manifest in the index. We also put the root
	// and the referrers of the manifests as the predecessors of copyRoot. This
	// is to ensure that all these nodes can be copied by calling extended copy.
	// Reference: https://github.com/oras-project/oras/issues/1728
	if len(rootReferrers) == 0 {
		copyRoot := index.Manifests[0]
		copyRootReferrers := append([]ocispec.Descriptor{root}, referrers...)
		findPredecessor := opts.FindPredecessors
		opts.FindPredecessors = func(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
			switch {
			case content.Equal(desc, root):
				return nil, nil
			case content.Equal(desc, copyRoot):
				return copyRootReferrers, nil
			}
			return findPredecessor(ctx, src, desc)
		}
		return opts, copyRoot, nil
	}

	rootReferrers = append(rootReferrers, referrers...)
	findPredecessor := opts.FindPredecessors
	opts.FindPredecessors = func(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		if content.Equal(desc, root) {
			return rootReferrers, nil
		}
		return findPredecessor(ctx, src, desc)
	}
	return opts, root, nil
}
