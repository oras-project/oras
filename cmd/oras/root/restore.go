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
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/command"
	"oras.land/oras/cmd/oras/internal/display"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

type restoreOptions struct {
	option.Common
	option.Remote
	option.Terminal

	// flags
	input            string
	excludeReferrers bool
	dryRun           bool
	concurrency      int

	// derived options
	repository string
	tags       []string
}

func restoreCmd() *cobra.Command {
	var opts restoreOptions
	cmd := &cobra.Command{
		Use:   "restore [flags] --input <path> <registry>/<repository>[:<ref1>[,<ref2>...]]",
		Short: "[Experimental] Restore artifacts to a registry from an OCI image layout",
		Long: `[Experimental] Restore artifacts to a registry from an OCI image layout, which can be a directory or a tar archive. 
If the input path ends with ".tar", it is recognized as a tar archive; otherwise, it is recognized as a directory.

Example - Restore artifacts from a backup file to a registry with multiple tags:
  oras restore localhost:5000/hello:v1,v2 --input hello-snapshot.tar

Example - Restore artifacts from a backup folder to a registry excluding referrers:
  oras restore localhost:5000/hello --input hello-snapshot --exclude-referrers

Example - Perform a dry run of the restore process without uploading artifacts:
  oras restore localhost:5000/hello:v1 --input hello-snapshot.tar --dry-run
`,
		Args: oerrors.CheckArgs(argument.Exactly(1), "the target repository to restore to"),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := option.Parse(cmd, &opts); err != nil {
				return err
			}
			if opts.input == "" {
				return errors.New("the input path cannot be empty")
			}

			// parse repo and tags
			var err error
			opts.repository, opts.tags, err = parseArtifactReferences(args[0])
			if err != nil {
				return err
			}

			opts.DisableTTY(opts.Debug, false)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Printer.Verbose = true // always print verbose output
			return runRestore(cmd, &opts)
		},
	}

	// required flag
	cmd.Flags().StringVar(&opts.input, "input", "", "restore from a folder or archive file to registry")
	_ = cmd.MarkFlagRequired("input")
	// optional flags
	cmd.Flags().BoolVar(&opts.excludeReferrers, "exclude-referrers", false, "restore the image from backup excluding referrers")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "simulate the restore process without actually uploading any artifacts to the target registry")
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

	// prepare the target registry
	dstRepo, err := opts.NewRepository(opts.repository, opts.Common, logger)
	if err != nil {
		return fmt.Errorf("failed to prepare target repository %q: %w", opts.repository, err)
	}
	statusHandler, metadataHandler := display.NewRestoreHandler(opts.Printer, opts.TTY, dstRepo)

	// prepare the source OCI store
	var srcOCI oras.ReadOnlyGraphTarget
	if strings.HasSuffix(opts.input, ".tar") {
		srcOCI, err = oci.NewFromTar(ctx, opts.input)
		if err != nil {
			return fmt.Errorf("failed to prepare OCI store from tar %q: %w", opts.input, err)
		}
		fi, err := os.Stat(opts.input)
		if err != nil {
			return fmt.Errorf("failed to stat tar file %q: %w", opts.input, err)
		}
		if err := metadataHandler.OnTarLoaded(opts.input, fi.Size()); err != nil {
			return err
		}
	} else {
		srcOCI, err = oci.NewWithContext(ctx, opts.input)
		if err != nil {
			return fmt.Errorf("failed to prepare OCI store from directory %q: %w", opts.input, err)
		}
	}

	// resolve tags to restore
	tags, roots, err := resolveTags(ctx, srcOCI, opts.tags)
	if err != nil {
		return err
	}
	if len(tags) == 0 {
		return &oerrors.Error{
			Err:            fmt.Errorf("no tags found in OCI layout %s", opts.input),
			Recommendation: fmt.Sprintf(`If you want to list available tags in %s, use "oras repo tags --oci-layout"`, opts.input),
		}
	}
	if err := metadataHandler.OnTagsFound(tags); err != nil {
		return err
	}

	// prepare copy options
	copyOpts := oras.DefaultCopyOptions
	copyOpts.Concurrency = opts.concurrency
	copyOpts.PreCopy = statusHandler.PreCopy
	copyOpts.PostCopy = statusHandler.PostCopy
	copyOpts.OnCopySkipped = statusHandler.OnCopySkipped
	extCopyGraphOpts := oras.ExtendedCopyGraphOptions{
		CopyGraphOptions: copyOpts.CopyGraphOptions,
		FindPredecessors: func(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
			return registry.Referrers(ctx, srcOCI, desc, "")
		},
	}
	for i, tag := range tags {
		var referrerCount int
		if !opts.excludeReferrers {
			// count referrers from source
			referrerCount, err = countReferrers(ctx, srcOCI, tag, roots[i], extCopyGraphOpts)
			if err != nil {
				return fmt.Errorf("failed to count referrers for tag %q: %w", tag, err)
			}
		}
		if opts.dryRun {
			if err := metadataHandler.OnArtifactPushed(true, tag, referrerCount); err != nil {
				return err
			}
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
				if _, err := oras.Copy(ctx, srcOCI, tag, trackedDst, tag, copyOpts); err != nil {
					return fmt.Errorf("failed to copy tag %q from %q to %q: %w", tag, opts.input, opts.repository, err)
				}
				return nil
			}
			if err := recursiveCopy(ctx, srcOCI, trackedDst, tag, roots[i], extCopyGraphOpts); err != nil {
				return fmt.Errorf("failed to restore tag %q from %q to %q: %w", tag, opts.input, opts.repository, err)
			}
			return nil
		}(); err != nil {
			return oerrors.UnwrapCopyError(err)
		}

		if err := metadataHandler.OnArtifactPushed(false, tag, referrerCount); err != nil {
			return err
		}
	}

	duration := time.Since(startTime)
	return metadataHandler.OnRestoreCompleted(opts.dryRun, len(tags), opts.repository, duration)
}
