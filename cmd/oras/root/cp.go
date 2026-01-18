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
	"oras.land/oras/cmd/oras/internal/display/metadata"
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

Example - Copy certain platforms of an artifact:
  oras cp --platform linux/amd64,linux/arm64,linux/arm/v7 localhost:5000/net-monitor:v1 localhost:6000/net-monitor-copy:v1

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

	// Check if multiple platforms are specified
	if len(opts.Platform.Platforms) > 1 && !opts.recursive {
		// Handle multiple platforms - copy manifests that match the specified platforms
		return copyMultiplePlatforms(ctx, statusHandler, metadataHandler, src, dst, opts)
	}

	// Handle single platform or recursive mode
	return copySinglePlatformOrRecursive(ctx, statusHandler, metadataHandler, src, dst, opts)
}

func copySinglePlatformOrRecursive(ctx context.Context, statusHandler status.CopyHandler, metadataHandler metadata.CopyHandler, src oras.ReadOnlyGraphTarget, dst oras.GraphTarget, opts *copyOptions) error {
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

// copyMultiplePlatforms handles copying when multiple platforms are specified
func copyMultiplePlatforms(ctx context.Context, statusHandler status.CopyHandler, metadataHandler metadata.CopyHandler, src oras.ReadOnlyGraphTarget, dst oras.GraphTarget, opts *copyOptions) error {
	// Resolve the source reference to get the root descriptor
	resolveOpts := oras.DefaultResolveOptions
	// We don't set TargetPlatform here since we want to get the full index/list
	root, err := oras.Resolve(ctx, src, opts.From.Reference, resolveOpts)
	if err != nil {
		return fmt.Errorf("failed to resolve %s: %w", opts.From.Reference, err)
	}

	// Check if the resolved descriptor is an index/manifest list
	isIndex := root.MediaType == ocispec.MediaTypeImageIndex || root.MediaType == docker.MediaTypeManifestList
	if !isIndex {
		// If not an index, return an error
		return fmt.Errorf("source reference %s is not an index or manifest list", opts.From.Reference)
	}

	// For indexes/lists, fetch the index content
	indexContent, err := content.FetchAll(ctx, src, root)
	if err != nil {
		return fmt.Errorf("failed to fetch index: %w", err)
	}

	var index ocispec.Index
	if err := json.Unmarshal(indexContent, &index); err != nil {
		return fmt.Errorf("failed to parse index: %w", err)
	}

	var availablePlatforms []string
	// Filter manifests based on the specified platforms
	var filteredManifests []ocispec.Descriptor
	matchedPlatforms := make(map[string]bool)
	for _, manifest := range index.Manifests {
		if manifest.Platform == nil {
			continue
		}
		availablePlatforms = append(availablePlatforms, fmt.Sprintf("%s/%s", manifest.Platform.OS, manifest.Platform.Architecture))
		if matchesAnyPlatform(manifest.Platform, opts.Platform.Platforms) {
			filteredManifests = append(filteredManifests, manifest)
			matchedPlatforms[fmt.Sprintf("%s/%s", manifest.Platform.OS, manifest.Platform.Architecture)] = true
		}
	}

	if len(filteredManifests) != len(opts.Platform.Platforms) {

		var unmatchedPlatforms []string
		for _, platform := range opts.Platform.Platforms {
			platformStr := fmt.Sprintf("%s/%s", platform.OS, platform.Architecture)
			if !matchedPlatforms[platformStr] {
				unmatchedPlatforms = append(unmatchedPlatforms, platformStr)
			}
		}

		// Return error with details about unmatched platforms
		return fmt.Errorf("only %d of %d requested platforms were matched: unmatched platforms: [%s]; available platforms in index: [%s]",
			len(filteredManifests), len(opts.Platform.Platforms), strings.Join(unmatchedPlatforms, ", "), strings.Join(availablePlatforms, ", "))
	}

	// Create a new index with only the filtered manifests
	newIndex := index
	newIndex.Manifests = filteredManifests

	// Marshal the new index
	newIndexContent, err := json.Marshal(newIndex)
	if err != nil {
		return fmt.Errorf("failed to marshal new index: %w", err)
	}

	// Create a descriptor for the new index
	newIndexDesc := ocispec.Descriptor{
		MediaType:   index.MediaType,
		Digest:      digest.FromBytes(newIndexContent),
		Size:        int64(len(newIndexContent)),
		Annotations: index.Annotations,
	}

	// Prepare copy options
	extendedCopyGraphOptions := oras.DefaultExtendedCopyGraphOptions
	extendedCopyGraphOptions.Concurrency = opts.concurrency
	extendedCopyGraphOptions.FindPredecessors = func(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		return registry.Referrers(ctx, src, desc, "")
	}

	if mountRepo, canMount := getMountPoint(src, dst, opts); canMount {
		extendedCopyGraphOptions.MountFrom = func(ctx context.Context, desc ocispec.Descriptor) ([]string, error) {
			return []string{mountRepo}, nil
		}
	}
	dst, err = statusHandler.StartTracking(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = statusHandler.StopTracking()
	}()
	extendedCopyGraphOptions.OnCopySkipped = statusHandler.OnCopySkipped
	extendedCopyGraphOptions.PreCopy = statusHandler.PreCopy
	extendedCopyGraphOptions.PostCopy = statusHandler.PostCopy
	extendedCopyGraphOptions.OnMounted = statusHandler.OnMounted

	// Copy all matching manifests and their content
	for _, manifestDesc := range filteredManifests {
		// Copy the manifest itself
		if err := oras.CopyGraph(ctx, src, dst, manifestDesc, extendedCopyGraphOptions.CopyGraphOptions); err != nil {
			return fmt.Errorf("failed to copy manifest %s: %w", manifestDesc.Digest, err)
		}
	}

	// Push the new index to the destination
	if err := dst.Push(ctx, newIndexDesc, strings.NewReader(string(newIndexContent))); err != nil {
		return fmt.Errorf("failed to push new index: %w", err)
	}

	// Tag the new index if needed
	if opts.To.Reference != "" {
		if err := dst.Tag(ctx, newIndexDesc, opts.To.Reference); err != nil {
			return fmt.Errorf("failed to tag new index: %w", err)
		}
	}

	// Handle extra references
	if len(opts.extraRefs) != 0 {
		tagNOpts := oras.DefaultTagNOptions
		tagNOpts.Concurrency = opts.concurrency
		tagListener := listener.NewTaggedListener(dst, metadataHandler.OnTagged)
		if _, err = oras.TagN(ctx, tagListener, opts.To.Reference, opts.extraRefs, tagNOpts); err != nil {
			return err
		}
	}

	// Update reference if needed
	if from, err := digest.Parse(opts.From.Reference); err == nil && from != newIndexDesc.Digest {
		opts.From.RawReference = fmt.Sprintf("%s@%s", opts.From.Path, newIndexDesc.Digest.String())
	}

	if err := metadataHandler.OnCopied(&opts.BinaryTarget, newIndexDesc); err != nil {
		return err
	}

	return metadataHandler.Render()
}

// matchesAnyPlatform checks if a manifest platform matches any of the specified platforms
func matchesAnyPlatform(manifestPlatform *ocispec.Platform, platforms []*ocispec.Platform) bool {
	for _, platform := range platforms {
		if platformMatches(manifestPlatform, platform) {
			return true
		}
	}
	return false
}

// platformMatches checks if two platforms match
func platformMatches(a, b *ocispec.Platform) bool {
	if a.OS != b.OS || a.Architecture != b.Architecture {
		return false
	}

	// Variant is optional; only treat it as a mismatch if both variants are non-empty and different.
	if a.Variant != "" && b.Variant != "" && a.Variant != b.Variant {
		return false
	}

	// OSVersion is optional; only treat it as a mismatch if both OSVersions are non-empty and different.
	if a.OSVersion != "" && b.OSVersion != "" && a.OSVersion != b.OSVersion {
		return false
	}

	return true
}

func doCopy(ctx context.Context, copyHandler status.CopyHandler, src oras.ReadOnlyGraphTarget, dst oras.GraphTarget, opts *copyOptions) (desc ocispec.Descriptor, err error) {
	// Prepare copy options
	extendedCopyGraphOptions := oras.DefaultExtendedCopyGraphOptions
	extendedCopyGraphOptions.Concurrency = opts.concurrency
	extendedCopyGraphOptions.FindPredecessors = func(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		return registry.Referrers(ctx, src, desc, "")
	}

	if mountRepo, canMount := getMountPoint(src, dst, opts); canMount {
		extendedCopyGraphOptions.MountFrom = func(ctx context.Context, desc ocispec.Descriptor) ([]string, error) {
			return []string{mountRepo}, nil
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

func prepareCopyOption(ctx context.Context, src oras.ReadOnlyGraphTarget, _ oras.Target, root ocispec.Descriptor, opts oras.ExtendedCopyGraphOptions) (oras.ExtendedCopyGraphOptions, ocispec.Descriptor, error) {
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

// getMountPoint checks if mounting can be performed between two targets and returns
// the repository name to be mounted from if applicable. Mount can be performed if the two
// targets are both remote repositories, are in the same registry and have identical credentials.
func getMountPoint(src oras.ReadOnlyGraphTarget, dst oras.GraphTarget, opts *copyOptions) (string, bool) {
	srcRepo, srcIsRemote := src.(*remote.Repository)
	dstRepo, dstIsRemote := dst.(*remote.Repository)
	if !srcIsRemote || !dstIsRemote {
		return "", false
	}
	if srcRepo.Reference.Registry != dstRepo.Reference.Registry {
		return "", false
	}
	srcCred := opts.From.Credential()
	dstCred := opts.To.Credential()
	if srcCred != dstCred {
		return "", false
	}
	return srcRepo.Reference.Repository, true
}
