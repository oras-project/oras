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
	"path/filepath"
	"regexp"
	"strings"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/command"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	orasio "oras.land/oras/internal/io"
)

type outputFormat int

const (
	outputFormatDir outputFormat = iota
	outputFormatTar
)

// tagRegexp matches valid OCI artifact tags.
// reference: https://github.com/opencontainers/distribution-spec/blob/v1.1.1/spec.md#pulling-manifests
var tagRegexp = regexp.MustCompile(`^[\w][\w.-]{0,127}$`)

type backupOptions struct {
	option.Common
	option.Remote
	option.Terminal

	// flags
	output           string
	includeReferrers bool
	concurrency      int

	// derived options
	outputFormat outputFormat
	repository   string
	tags         []string
}

func backupCmd() *cobra.Command {
	var opts backupOptions
	cmd := &cobra.Command{
		Use:   "backup [flags] --output <path> <registry>/<repository>[:<ref1>[,<ref2>...]]",
		Short: "[Experimental] Back up artifacts from a registry into an OCI image layout",
		Long: `[Experimental] Back up artifacts from a registry into an OCI image layout, saved either as a directory or a tar archive.
The output format is determined by the file extension of the specified output path: if it ends with ".tar", the output will be a tar archive; otherwise, it will be a directory.

Example - Back up an artifact and its referrers to a directory:
  oras backup --output hello --include-referrers localhost:5000/hello:v1

Example - Back up an artifact and its referrers to a tar archive:
  oras backup --output hello.tar --include-referrers localhost:5000/hello:v1

Example - Back up multiple tagged artifacts and their referrers:
  oras backup --output hello --include-referrers localhost:5000/hello:v1,v2,v3

Example - Back up all tagged artifacts and their referrers in a repository:
  oras backup --output hello --include-referrers localhost:5000/hello

Example - Back up an artifact and its referrers discovered via Referrers API:
  oras backup --output hello --include-referrers --distribution-spec v1.1-referrers-api localhost:5000/hello

Example - Back up an artifact and its referrers discovered via Referrers Tag Schema:
  oras backup --output hello --include-referrers --distribution-spec v1.1-referrers-tag localhost:5000/hello

Example - Back up from an insecure registry:
  oras backup --output hello.tar --insecure localhost:5000/hello:v1

Example - Back up from a plain HTTP registry:
  oras backup --output hello.tar --plain-http localhost:5000/hello:v1

Example - Back up with a custom concurrency level:
  oras backup --output hello.tar --concurrency 6 localhost:5000/hello:v1
`,
		Args: oerrors.CheckArgs(argument.Exactly(1), "the artifacts to back up"),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := option.Parse(cmd, &opts); err != nil {
				return err
			}
			if opts.output == "" {
				return errors.New("the output path cannot be empty")
			}

			// parse repo and references
			var err error
			opts.repository, opts.tags, err = parseArtifactReferences(args[0])
			if err != nil {
				return err
			}

			// parse output format
			if strings.HasSuffix(opts.output, ".tar") {
				opts.outputFormat = outputFormatTar
			} else {
				opts.outputFormat = outputFormatDir
			}

			opts.DisableTTY(opts.Debug, false)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Printer.Verbose = true // always print verbose output
			return runBackup(cmd, &opts)
		},
	}

	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "path to the target output, either a tar archive (*.tar) or a directory")
	cmd.Flags().BoolVarP(&opts.includeReferrers, "include-referrers", "", false, "back up the artifact with its referrers (e.g., attestations, SBOMs)")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 3, "concurrency level")
	opts.EnableDistributionSpecFlag()
	_ = cmd.MarkFlagRequired("output")
	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.Remote)
}

func runBackup(cmd *cobra.Command, opts *backupOptions) error {
	startTime := time.Now() // start timing the backup process
	ctx, logger := command.GetLogger(cmd, &opts.Common)

	var dstRoot string
	switch opts.outputFormat {
	case outputFormatDir:
		dstRoot = opts.output
	case outputFormatTar:
		tempDir, err := os.MkdirTemp("", "oras-backup-*")
		if err != nil {
			return fmt.Errorf("failed to create temporary directory for backup: %w", err)
		}
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				logger.Debugf("failed to remove temporary directory %s: %v", tempDir, err)
			}
		}()
		dstRoot = tempDir
	default:
		// this should not happen, just a safeguard
		return fmt.Errorf("unsupported output format")
	}

	// Prepare copy source and destination
	srcRepo, err := opts.NewRepository(opts.repository, opts.Common, logger)
	if err != nil {
		return fmt.Errorf("failed to prepare repository %s for backup: %w", opts.repository, err)
	}
	dstOCI, err := oci.New(dstRoot)
	if err != nil {
		return fmt.Errorf("failed to prepare OCI store for backup: %w", err)
	}
	statusHandler, metadataHandler := display.NewBackupHandler(opts.Printer, opts.TTY, opts.repository, dstOCI)

	// Resolve tags to back up
	tags, roots, err := resolveTags(ctx, srcRepo, opts.tags)
	if err != nil {
		return err
	}
	if len(tags) == 0 {
		return &oerrors.Error{
			Err:            fmt.Errorf("no tags found in repository %s", opts.repository),
			Recommendation: fmt.Sprintf(`If you want to list available tags in %s, use "oras repo tags"`, opts.repository),
		}
	}
	if err := metadataHandler.OnTagsFound(tags); err != nil {
		return err
	}

	// Prepare copy options
	copyGraphOpts := oras.DefaultCopyGraphOptions
	copyGraphOpts.Concurrency = opts.concurrency
	copyGraphOpts.PreCopy = statusHandler.PreCopy
	copyGraphOpts.PostCopy = statusHandler.PostCopy
	copyGraphOpts.OnCopySkipped = statusHandler.OnCopySkipped
	extCopyGraphOpts := oras.ExtendedCopyGraphOptions{
		CopyGraphOptions: copyGraphOpts,
	}

	for i, t := range tags {
		referrerCount, err := func(tag string) (referrerCount int, retErr error) {
			trackedDst, err := statusHandler.StartTracking(dstOCI)
			if err != nil {
				return 0, err
			}
			defer func() {
				stopErr := statusHandler.StopTracking()
				if retErr == nil {
					retErr = stopErr
				}
			}()

			return backupTag(ctx, srcRepo, trackedDst, roots[i], t, opts.includeReferrers, copyGraphOpts, extCopyGraphOpts)
		}(t)
		if err != nil {
			return oerrors.UnwrapCopyError(err)
		}
		if err := metadataHandler.OnArtifactPulled(t, referrerCount); err != nil {
			return err
		}
	}

	if err := prepareBackupOutput(ctx, dstRoot, opts, logger, metadataHandler); err != nil {
		return err
	}
	duration := time.Since(startTime)
	return metadataHandler.OnBackupCompleted(len(tags), opts.output, duration)
}

func backupTag(ctx context.Context,
	src oras.ReadOnlyGraphTarget,
	dst oras.GraphTarget,
	root ocispec.Descriptor,
	tag string,
	includeReferrers bool,
	copyGraphOpts oras.CopyGraphOptions,
	extCopyGraphOpts oras.ExtendedCopyGraphOptions) (int, error) {
	if !includeReferrers {
		if err := oras.CopyGraph(ctx, src, dst, root, copyGraphOpts); err != nil {
			return 0, fmt.Errorf("failed to pull tag %q, digest %q: %w", tag, root.Digest.String(), err)
		}
		if err := dst.Tag(ctx, root, tag); err != nil {
			return 0, fmt.Errorf("failed to tag %q with %q: %w", root.Digest.String(), tag, err)
		}
		return 0, nil
	}

	// copy with referrers
	if err := recursiveCopy(ctx, src, dst, tag, root, extCopyGraphOpts); err != nil {
		return 0, fmt.Errorf("failed to pull tag %q and referrers, digest %q: %w", tag, root.Digest.String(), err)
	}
	referrers, err := registry.Referrers(ctx, dst, root, "")
	if err != nil {
		return 0, fmt.Errorf("failed to get referrers for tag %q, digest %q: %w", tag, root.Digest.String(), err)
	}
	return len(referrers), nil
}

func prepareBackupOutput(ctx context.Context, dstRoot string, opts *backupOptions, logger logrus.FieldLogger, metadataHandler metadata.BackupHandler) error {
	// Remove ingest dir for a cleaner output
	ingestDir := filepath.Join(dstRoot, "ingest")
	if _, err := os.Stat(ingestDir); err == nil {
		if err := os.RemoveAll(ingestDir); err != nil {
			logger.Debugf("failed to remove ingest directory: %v", err)
		}
	}
	if opts.outputFormat != outputFormatTar {
		// If output format is not a tar, we are done
		return nil
	}

	if err := metadataHandler.OnTarExporting(opts.output); err != nil {
		return err
	}
	// Create a temporary file for the tarball
	tempTar, err := os.CreateTemp("", "oras-backup-*.tar")
	if err != nil {
		return fmt.Errorf("failed to create temporary tar file: %w", err)
	}
	tempTarPath := tempTar.Name()
	if err := orasio.TarDirectory(ctx, tempTar, dstRoot); err != nil {
		return fmt.Errorf("failed to create tar archive from directory %s: %w", dstRoot, err)
	}
	if err := tempTar.Close(); err != nil {
		return fmt.Errorf("failed to close temporary tar file: %w", err)
	}

	// Ensure target directory exists
	absOutput := opts.output
	if !filepath.IsAbs(absOutput) {
		absOutput, err = filepath.Abs(opts.output)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for output file %s: %w", opts.output, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(absOutput), 0755); err != nil {
		return fmt.Errorf("failed to create directory for output file %s: %w", absOutput, err)
	}

	// Move the temporary tar file to the final output path
	if err := os.Rename(tempTarPath, absOutput); err != nil {
		removeErr := os.Remove(tempTarPath)
		if removeErr != nil {
			logger.Debugf("failed to remove temporary tar file %s: %v", tempTarPath, removeErr)
		}
		return err
	}

	fi, err := os.Stat(absOutput)
	if err != nil {
		return fmt.Errorf("failed to stat output file %s: %w", absOutput, err)
	}
	return metadataHandler.OnTarExported(opts.output, fi.Size())
}

// resolveTags resolves tags to their descriptors.
// It returns the resolved tags and their corresponding descriptors.
func resolveTags(ctx context.Context, target oras.ReadOnlyTarget, specifiedTags []string) ([]string, []ocispec.Descriptor, error) {
	var tags []string
	var descs []ocispec.Descriptor
	if len(specifiedTags) > 0 {
		// resolve the specified tags
		tags = specifiedTags
		descs = make([]ocispec.Descriptor, 0, len(tags))
		for _, tag := range tags {
			desc, err := oras.Resolve(ctx, target, tag, oras.DefaultResolveOptions)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to resolve tag %q: %w", tag, err)
			}
			descs = append(descs, desc)
		}
		return tags, descs, nil
	}

	// discover all tags in the repository and resolve them
	tagLister, ok := target.(registry.TagLister)
	if !ok {
		return nil, nil, errors.New("the target does not support tag listing")
	}
	if err := tagLister.Tags(ctx, "", func(gotTags []string) error {
		for _, gotTag := range gotTags {
			desc, err := oras.Resolve(ctx, target, gotTag, oras.DefaultResolveOptions)
			if err != nil {
				return fmt.Errorf("failed to resolve tag %q: %w", gotTag, err)
			}
			tags = append(tags, gotTag)
			descs = append(descs, desc)
		}
		return nil
	}); err != nil {
		return nil, nil, fmt.Errorf("failed to find tags: %w", err)
	}

	return tags, descs, nil
}

func parseArtifactReferences(artifactRefs string) (repository string, tags []string, err error) {
	// Validate input
	if artifactRefs == "" {
		return "", nil, errors.New("artifact reference cannot be empty")
	}
	// Reject digest references early
	if strings.ContainsRune(artifactRefs, '@') {
		return "", nil, fmt.Errorf("digest references are not supported: %q", artifactRefs)
	}

	// 1. Split the input into repository and tag parts
	lastSlash := strings.LastIndexByte(artifactRefs, '/')
	lastColon := strings.LastIndexByte(artifactRefs, ':')

	var repoParts string
	var tagsPart string
	if lastColon != -1 && lastColon > lastSlash {
		// A colon after the last slash denotes the beginning of tags
		repoParts = artifactRefs[:lastColon]
		tagsPart = artifactRefs[lastColon+1:]
	} else {
		repoParts = artifactRefs
		// tagPart stays empty - no tags
	}

	// 2. Validate repository
	parsedRepo, err := registry.ParseReference(repoParts)
	if err != nil {
		return "", nil, fmt.Errorf("invalid repository %q in reference %q: %w", repoParts, artifactRefs, err)
	}
	repository = parsedRepo.String()

	// 3. Process tags
	if tagsPart == "" {
		return repository, nil, nil
	}
	tagList := strings.Split(tagsPart, ",")
	tags = make([]string, 0, len(tagList))

	// Validate each tag
	for _, tag := range tagList {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue // skip empty tags
		}
		if !tagRegexp.MatchString(tag) {
			return "", nil, fmt.Errorf("invalid tag %q in reference %q: tag must match %s", tag, artifactRefs, tagRegexp)
		}
		tags = append(tags, tag)
	}
	return repository, tags, nil
}
