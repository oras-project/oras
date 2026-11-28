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
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/oras-project/oras-go/v3"
	"github.com/oras-project/oras-go/v3/content"
	"github.com/oras-project/oras-go/v3/content/oci"
	"github.com/oras-project/oras-go/v3/errdef"
	"github.com/oras-project/oras-go/v3/registry"
	"github.com/oras-project/oras-go/v3/registry/remote/properties"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/command"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/status"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/descriptor"
	orasio "oras.land/oras/internal/io"
)

type restoreOptions struct {
	option.Common
	option.Remote
	option.Terminal
	option.Concurrency

	// flags
	input                  string
	excludeReferrers       bool
	dryRun                 bool
	preserveSourceRegistry bool

	// derived options
	target string // target registry, optionally with a namespace
}

func restoreCmd() *cobra.Command {
	var opts restoreOptions
	cmd := &cobra.Command{
		Use:   "restore [flags] --input <path> <registry>[/<namespace>]",
		Short: "[Experimental] Restore artifacts to a registry from an OCI image layout",
		Long: `[Experimental] Restore artifacts to a registry from an OCI image layout, which can be either a directory or a tar archive.

Each artifact is restored to <registry>[/<namespace>]/<repository>, where <repository> is the repository path
recorded in the backup. The source registry is dropped unless --preserve-source-registry is used.
Backups created by earlier versions record only tags, so they are restored to the given <registry>/<repository>.

Example - Restore all tagged artifacts from a tar archive:
  oras restore --input hello.tar localhost:5000

Example - Restore all tagged artifacts from a directory under a namespace:
  oras restore --input hello localhost:5000/backup

Example - Perform a dry run without actually uploading artifacts:
  oras restore --input hello --dry-run localhost:5000

Example - Keep the source registry in the target repository path, e.g. restore localhost:5001/app to localhost:5000/localhost_5001/app:
  oras restore --input hello --preserve-source-registry localhost:5000

Example - Exclude referrers when restoring artifacts:
  oras restore --input hello --exclude-referrers localhost:5000

Example - Use Referrers API for discovering referrers:
  oras restore --input hello --distribution-spec v1.1-referrers-api localhost:5000

Example - Use Referrers Tag Schema for discovering referrers:
  oras restore --input hello --distribution-spec v1.1-referrers-tag localhost:5000

Example - Restore to an insecure registry:
  oras restore --input hello --insecure localhost:5000

Example - Restore to a plain HTTP registry (no TLS):
  oras restore --input hello --plain-http localhost:5000

Example - Set custom concurrency level:
  oras restore --input hello --concurrency 6 localhost:5000
`,
		Args: oerrors.CheckArgs(argument.Exactly(1), "the targets to restore to"),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := option.Parse(cmd, &opts); err != nil {
				return err
			}

			// parse target
			var err error
			opts.target, err = parseRestoreTarget(args[0])
			if err != nil {
				return err
			}

			opts.DisableTTY(opts.Debug, false)
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.Printer.Verbose = true // always print verbose output
			return runRestore(cmd, &opts)
		},
	}

	// required flag
	cmd.Flags().StringVar(&opts.input, "input", "", "path to the OCI layout, either a tar archive (*.tar) or a directory")
	_ = cmd.MarkFlagRequired("input")
	// optional flags
	cmd.Flags().BoolVar(&opts.excludeReferrers, "exclude-referrers", false, "restore artifacts excluding their referrers")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "simulate the restore process without actually uploading any artifacts")
	cmd.Flags().BoolVar(&opts.preserveSourceRegistry, "preserve-source-registry", false, "keep the source registry as the first path component of each target repository, lowercased and with ':' replaced by '_' (e.g. localhost_5000)")
	opts.Concurrency.Concurrency = 5
	opts.EnableDistributionSpecFlag()
	// apply flags
	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.Remote)
}

func runRestore(cmd *cobra.Command, opts *restoreOptions) error {
	if opts.input == "" {
		return errors.New("the input path cannot be empty")
	}
	startTime := time.Now() // start timing the restore process
	ctx, logger := command.GetLogger(cmd, &opts.Common)

	// prepare the target repositories on demand
	newTarget := func(repo string) (oras.GraphTarget, error) {
		return opts.NewRepository(repo, opts.Common, logger)
	}
	statusHandler, metadataHandler := display.NewRestoreHandler(opts.Printer, opts.TTY, nil, opts.dryRun)
	return doRestore(ctx, opts, logger, newTarget, startTime, statusHandler, metadataHandler)
}

func doRestore(
	ctx context.Context,
	opts *restoreOptions,
	logger logrus.FieldLogger,
	newTarget func(repo string) (oras.GraphTarget, error),
	startTime time.Time,
	statusHandler status.RestoreHandler,
	metadataHandler metadata.RestoreHandler,
) error {
	// prepare the source OCI store
	var srcOCI oras.ReadOnlyGraphTarget
	fi, err := os.Stat(opts.input)
	if err != nil {
		return fmt.Errorf("failed to access input path %q: %w", opts.input, err)
	}
	switch {
	case fi.Mode().IsRegular():
		isTar, err := orasio.IsTarFile(opts.input)
		if err != nil {
			return fmt.Errorf("unable to determine if %q is a tar archive: %w", opts.input, err)
		}
		if !isTar {
			return fmt.Errorf("input path %q is not a tar archive", opts.input)
		}
		srcOCI, err = oci.NewFromTar(ctx, opts.input)
		if err != nil {
			return fmt.Errorf("failed to prepare OCI store from tar archive %q: %w", opts.input, err)
		}
		if err := metadataHandler.OnTarLoaded(opts.input, fi.Size()); err != nil {
			return err
		}
	case fi.IsDir():
		srcOCI, err = oci.NewWithContext(ctx, opts.input)
		if err != nil {
			return fmt.Errorf("failed to prepare OCI store from directory %q: %w", opts.input, err)
		}
	default:
		return fmt.Errorf("input path %q must be a directory or a tar archive", opts.input)
	}

	// resolve tags to restore
	srcRefs, tags, roots, err := resolveRestoreRefs(ctx, srcOCI)
	if err != nil {
		return err
	}
	if len(tags) == 0 {
		return &oerrors.Error{
			Err:            fmt.Errorf("no tags found in OCI layout %q", opts.input),
			Recommendation: fmt.Sprintf(`If you want to list available tags in %q, use "oras repo tags --oci-layout"`, opts.input),
		}
	}
	// map each artifact to its target repository before pushing anything
	dstRepos, err := restoreTargets(opts.target, srcRefs, opts.preserveSourceRegistry)
	if err != nil {
		return err
	}
	if err := metadataHandler.OnTagsFound(tags); err != nil {
		return err
	}

	// prepare copy options
	copyOpts := oras.DefaultCopyOptions
	copyOpts.Concurrency = opts.Concurrency.Concurrency
	copyOpts.PreCopy = statusHandler.PreCopy
	copyOpts.PostCopy = statusHandler.PostCopy
	copyOpts.OnCopySkipped = statusHandler.OnCopySkipped
	extCopyGraphOpts := oras.ExtendedCopyGraphOptions{
		CopyGraphOptions: copyOpts.CopyGraphOptions,
		FindPredecessors: func(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
			return registry.Referrers(ctx, src, desc, "")
		},
	}
	targets := make(map[string]oras.GraphTarget)
	for i, tag := range tags {
		dstRef := dstRepos[i] + ":" + tag
		var referrerCount int
		if !opts.excludeReferrers {
			// count referrers from source
			referrerCount, err = countReferrers(ctx, srcOCI, tag, roots[i], extCopyGraphOpts)
			if err != nil {
				return fmt.Errorf("failed to count referrers for tag %q: %w", tag, err)
			}
		}
		dstRepo, ok := targets[dstRepos[i]]
		if !ok {
			dstRepo, err = newTarget(dstRepos[i])
			if err != nil {
				return fmt.Errorf("failed to prepare target repository %q: %w", dstRepos[i], err)
			}
			targets[dstRepos[i]] = dstRepo
		}
		warnIfReplacingIndex(ctx, dstRepo, tag, roots[i], logger)
		if opts.dryRun {
			if err := metadataHandler.OnArtifactPushed(srcRefs[i], dstRef, referrerCount); err != nil {
				return err
			}
			// dry run, skip actual copy
			continue
		}
		if err := func() (retErr error) {
			trackedDst, err := statusHandler.StartTracking(dstRepo)
			if err != nil {
				return err
			}
			defer func() {
				stopErr := statusHandler.StopTracking()
				if retErr == nil {
					retErr = stopErr
				}
			}()

			if opts.excludeReferrers {
				_, err := oras.Copy(ctx, srcOCI, srcRefs[i], trackedDst, tag, copyOpts)
				return err
			}
			return recursiveCopy(ctx, srcOCI, trackedDst, tag, roots[i], extCopyGraphOpts)
		}(); err != nil {
			return fmt.Errorf("failed to restore tag %q from %q to %q: %w", tag, opts.input, dstRepos[i], oerrors.UnwrapCopyError(err))
		}

		if err := metadataHandler.OnArtifactPushed(srcRefs[i], dstRef, referrerCount); err != nil {
			return err
		}
	}

	duration := time.Since(startTime)
	return metadataHandler.OnRestoreCompleted(len(tags), opts.target, duration)
}

// warnIfReplacingIndex warns when restoring a single manifest would narrow an
// existing multi-platform tag. Resolving the destination is advisory only.
func warnIfReplacingIndex(ctx context.Context, dst oras.ReadOnlyTarget, tag string, root ocispec.Descriptor, logger logrus.FieldLogger) {
	existing, err := dst.Resolve(ctx, tag)
	if errors.Is(err, errdef.ErrNotFound) {
		return
	}
	if err != nil {
		logger.Debugf("failed to resolve destination tag %q: %v", tag, err)
		return
	}
	if existing.Digest != root.Digest && descriptor.IsIndex(existing) && !descriptor.IsIndex(root) {
		logger.Warnf("tag %q currently points to multi-platform index %s; restoring replaces it with a single-platform manifest", tag, existing.Digest)
	}
}

// resolveRestoreRefs resolves the artifacts to restore from the backup in src.
// For each artifact it returns the reference stored in the backup, the tag to
// restore it as, and its descriptor.
//
// Backups store the full reference (<registry>/<repository>:<tag>) in
// org.opencontainers.image.ref.name. Backups created by earlier versions store
// only the tag.
func resolveRestoreRefs(ctx context.Context, src oras.ReadOnlyTarget) (srcRefs, tags []string, roots []ocispec.Descriptor, err error) {
	srcRefs, roots, err = resolveTags(ctx, src, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	tags = make([]string, len(srcRefs))
	for i, ref := range srcRefs {
		if tags[i], err = tagFromRefName(ref); err != nil {
			return nil, nil, nil, err
		}
	}
	return srcRefs, tags, roots, nil
}

// restoreTargets returns the repository to restore each stored reference to.
//
// A full reference <registry>/<repository>:<tag> is restored to
// <target>/<repository>, or to <target>/<registry>/<repository> if
// preserveSourceRegistry is set, with <registry> encoded by
// encodeRegistryComponent. A tag-only reference from a backup created by
// an earlier version records no repository, so it is restored to target.
// Two source repositories restoring to the same repository fail before
// anything is pushed.
func restoreTargets(target string, srcRefs []string, preserveSourceRegistry bool) ([]string, error) {
	dstRepos := make([]string, len(srcRefs))
	sources := make(map[string]string, len(srcRefs)) // target repository -> source repository
	reported := make(map[string]bool)                // colliding source repositories already listed
	var collisions []string
	for i, srcRef := range srcRefs {
		if isTagOnlyRefName(srcRef) {
			if !strings.Contains(target, "/") {
				return nil, &oerrors.Error{
					Err:            fmt.Errorf("tag %q in the backup has no repository to restore to %q", srcRef, target),
					Recommendation: "Backups created by earlier versions record only tags. Specify the target as <registry>/<repository>",
				}
			}
			dstRepos[i] = target
			continue
		}
		ref, err := properties.NewReference(srcRef)
		if err != nil {
			return nil, fmt.Errorf("invalid reference %q in the backup: %w", srcRef, err)
		}
		srcRepo := ref.Registry + "/" + ref.Repository
		dstRepo := target + "/" + ref.Repository
		if preserveSourceRegistry {
			host, err := encodeRegistryComponent(ref.Registry)
			if err != nil {
				return nil, &oerrors.Error{
					Err:            fmt.Errorf("source registry of %q in the backup cannot be a repository path component: %w", srcRef, err),
					Recommendation: "Restore without --preserve-source-registry",
				}
			}
			dstRepo = target + "/" + host + "/" + ref.Repository
		}
		dst, err := properties.NewReference(dstRepo)
		if err != nil {
			return nil, fmt.Errorf("invalid target repository %q for %q in the backup: %w", dstRepo, srcRef, err)
		}
		if len(dst.Repository) > maxRepositoryLength {
			return nil, fmt.Errorf("target repository %q for %q in the backup is longer than %d characters", dstRepo, srcRef, maxRepositoryLength)
		}
		if prev, ok := sources[dstRepo]; ok && prev != srcRepo {
			if !reported[srcRepo] {
				reported[srcRepo] = true
				collisions = append(collisions, fmt.Sprintf("%q and %q both restore to %q", prev, srcRepo, dstRepo))
			}
			continue
		}
		sources[dstRepo] = srcRepo
		dstRepos[i] = dstRepo
	}
	if len(collisions) > 0 {
		return nil, &oerrors.Error{
			Err:            fmt.Errorf("source repositories collide in the target: %s", strings.Join(collisions, "; ")),
			Recommendation: "Use --preserve-source-registry to keep the source registry in the target repository path",
		}
	}
	return dstRepos, nil
}

// parseRestoreTarget parses the restore target <registry>[/<namespace>].
// A target without a path is a registry, where a ':' separates the port.
// Tags and digests are rejected: the target only names where artifacts go.
func parseRestoreTarget(target string) (string, error) {
	if !strings.Contains(target, "/") {
		if err := (properties.Reference{Registry: target}).ValidateRegistry(); err != nil {
			return "", fmt.Errorf("invalid target %q: %w", target, err)
		}
		return target, nil
	}
	if path := target[strings.Index(target, "/")+1:]; strings.ContainsAny(path, ":@") {
		return "", &oerrors.Error{
			Err:            fmt.Errorf("invalid target %q: a tag or digest cannot be specified in the restore target", target),
			Recommendation: "Specify the target as <registry>[/<namespace>]. Restore copies every artifact in the backup",
		}
	}
	if _, err := properties.NewReference(target); err != nil {
		return "", fmt.Errorf("invalid target %q: %w", target, err)
	}
	return target, nil
}

// maxRepositoryLength is the maximum length of a repository name accepted by
// registries.
const maxRepositoryLength = 255

// encodeRegistryComponent encodes a registry host as a repository path
// component: the host is lowercased and the ':' before a port is replaced by
// '_', e.g. registry.lan:5000 becomes registry.lan_5000. Host names cannot
// contain '_', so two different hosts never encode to the same component.
// IPv6 literal hosts cannot be encoded.
func encodeRegistryComponent(host string) (string, error) {
	if strings.HasPrefix(host, "[") {
		return "", fmt.Errorf("IPv6 literal host %q is not supported", host)
	}
	return strings.ReplaceAll(strings.ToLower(host), ":", "_"), nil
}

// isTagOnlyRefName reports whether an org.opencontainers.image.ref.name value
// is a tag only, as written by backups created by earlier versions.
func isTagOnlyRefName(name string) bool {
	return !strings.ContainsAny(name, "/:@")
}

// tagFromRefName returns the tag in an org.opencontainers.image.ref.name value
// written by oras backup. The value is either a full reference such as
// localhost:5000/repo:v1, or just a tag for backups created by earlier versions.
func tagFromRefName(name string) (string, error) {
	if isTagOnlyRefName(name) {
		// tag-only reference from a backup created by an earlier version
		if err := (properties.Reference{Tag: name}).ValidateTag(); err != nil {
			return "", fmt.Errorf("invalid tag %q in the backup: %w", name, err)
		}
		return name, nil
	}
	ref, err := properties.NewReference(name)
	if err != nil {
		return "", fmt.Errorf("invalid reference %q in the backup: %w", name, err)
	}
	if ref.Tag == "" || ref.Digest != "" {
		return "", fmt.Errorf("reference %q in the backup must have a tag and no digest: %w", name, errdef.ErrInvalidReference)
	}
	return ref.Tag, nil
}
