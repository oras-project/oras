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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
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

// tagRegexp checks the tag name.
// The docker and OCI spec have the same regular expression.
//
// Reference: https://github.com/opencontainers/distribution-spec/blob/v1.1.1/spec.md#pulling-manifests
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
		Short: "Back up artifacts from a registry into an OCI image layout, saved either as a directory or a tar archive",
		Long: `Back up artifacts from a registry into an OCI image layout, saved either as a directory or a tar archive. The output format is determined by the file extension of the specified output path: if it ends with ".tar", the output will be a tar archive; otherwise, it will be a directory.

Example - Back up an artifact with referrers from a registry to an OCI image layout directory:
  oras backup --output hello --include-referrers localhost:5000/hello:v1

Example - Back up an artifact with referrers from a registry to a tar archive:
  oras backup --output hello.tar --include-referrers localhost:5000/hello:v1

Example - Back up multiple artifacts with their referrers:
  oras backup --output hello.tar --include-referrers localhost:5000/hello:v1,v2,v3

Example - Back up artifact from an insecure registry:
  oras backup --output hello.tar --insecure localhost:5000/hello:v1

Example - Back up artifact from the HTTP registry:
  oras backup --output hello.tar --plain-http localhost:5000/hello:v1

Example - Back up with concurrency level tuned:
  oras backup --output hello.tar --concurrency 6 localhost:5000/hello:v1
`,
		Args: oerrors.CheckArgs(argument.Exactly(1), "the artifact reference you want to back up"),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := option.Parse(cmd, &opts); err != nil {
				return err
			}

			// parse repo and references
			var err error
			opts.repository, opts.tags, err = parseArtifactsToBackup(args[0])
			if err != nil {
				return err
			}

			// TODO: should we record abs file path of the output?
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
			return runBackup(cmd, &opts)
		},
	}

	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "target directory path or tar file path to write in local filesystem (required)")
	cmd.Flags().BoolVarP(&opts.includeReferrers, "include-referrers", "", false, "back up the image and its linked referrers (e.g., attestations, SBOMs)")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 3, "concurrency level")

	// Mark output flag as required
	_ = cmd.MarkFlagRequired("output")

	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.Remote)
}

func runBackup(cmd *cobra.Command, opts *backupOptions) error {
	ctx, logger := command.GetLogger(cmd, &opts.Common)

	// TODO: might need to refactor output format handling here
	var dstRoot string
	switch opts.outputFormat {
	case outputFormatDir:
		dstRoot = opts.output
	case outputFormatTar:
		tempDir, err := os.MkdirTemp("", "oras-backup-*")
		if err != nil {
			// TODO: better error message?
			return fmt.Errorf("failed to create temporary directory: %w", err)
		}
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				logger.Debugf("failed to remove temporary directory %s: %v", tempDir, err)
			}
		}()
		dstRoot = tempDir
	default:
		// this should not happen
		return fmt.Errorf("unsupported output format")
	}

	// Prepare remote srcRepo as the source
	srcRepo, err := opts.Remote.NewRepository(opts.repository, opts.Common, logger)
	if err != nil {
		return err
	}
	// Prepare OCI layout as the destination
	dstOCI, err := oci.New(dstRoot)
	if err != nil {
		return fmt.Errorf("failed to create OCI store: %w", err)
	}

	statusHandler, metadataHandler := display.NewBackupHandler(opts.Printer, opts.TTY, opts.repository)

	tags, err := findTagsToBackup(ctx, srcRepo, opts)
	if err != nil {
		return fmt.Errorf("failed to get tags to back up: %w", err)
	}
	if len(tags) == 0 {
		// TODO: better error message
		return fmt.Errorf("no tags to back up, please specify at least one tag")
	}
	metadataHandler.OnTagsFound(tags)

	// TODO: more options
	copyGraphOpts := oras.DefaultCopyGraphOptions
	copyGraphOpts.Concurrency = opts.concurrency
	copyGraphOpts.PreCopy = statusHandler.PreCopy
	copyGraphOpts.PostCopy = statusHandler.PostCopy
	copyGraphOpts.OnCopySkipped = statusHandler.OnCopySkipped
	// Do the backup
	copyOpts := oras.CopyOptions{
		CopyGraphOptions: copyGraphOpts,
	}
	extendedCopyOpts := oras.ExtendedCopyOptions{
		ExtendedCopyGraphOptions: oras.ExtendedCopyGraphOptions{
			CopyGraphOptions: copyGraphOpts,
		},
	}

	trackedDst, err := statusHandler.StartTracking(dstOCI)
	if err != nil {
		return err
	}
	defer func() {
		stopErr := statusHandler.StopTracking()
		if err == nil {
			err = stopErr
		}
	}()

	// TODO: handle copy error
	for _, tag := range tags {
		if opts.includeReferrers {
			desc, err := oras.Resolve(ctx, srcRepo, tag, oras.DefaultResolveOptions)
			if err != nil {
				return fmt.Errorf("failed to resolve %s: %w", tag, err)
			}
			extendedCopyOpts, err = prepareCopyOption(ctx, srcRepo, trackedDst, desc, extendedCopyOpts)
			if err != nil {
				return fmt.Errorf("failed to prepare extended copy options for %s: %w", tag, err)
			}
			_, err = oras.ExtendedCopy(ctx, srcRepo, desc.Digest.String(), trackedDst, tag, extendedCopyOpts)
			if err != nil {
				return fmt.Errorf("failed to copy tag %s: %w", tag, err)
			}

			referrers, err := registry.Referrers(ctx, dstOCI, desc, "")
			if err != nil {
				return fmt.Errorf("failed to get referrers for %s: %w", tag, err)
			}
			metadataHandler.OnArtifactPulled(tag, len(referrers))
		} else {
			_, err := oras.Copy(ctx, srcRepo, tag, trackedDst, tag, copyOpts)
			if err != nil {
				return fmt.Errorf("failed to copy ref %s: %w", tag, err)
			}
			metadataHandler.OnArtifactPulled(tag, 0)
		}
	}

	if err := prepareBackupOutput(ctx, dstRoot, opts, logger, metadataHandler); err != nil {
		return err
	}
	metadataHandler.OnBackupCompleted(len(tags), opts.output)
	return nil
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

	metadataHandler.OnExporting(opts.output)
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
	if err := os.MkdirAll(filepath.Dir(opts.output), 0777); err != nil {
		return fmt.Errorf("failed to create directory for output file %s: %w", opts.output, err)
	}

	// Move the temporary tar file to the final output path
	if err := os.Rename(tempTarPath, opts.output); err != nil {
		removeErr := os.Remove(tempTarPath)
		if removeErr != nil {
			logger.Debugf("failed to remove temporary tar file %s: %v", tempTarPath, removeErr)
		}
		return err
	}

	metadataHandler.OnExported(opts.output)
	return nil
}

func findTagsToBackup(ctx context.Context, repo *remote.Repository, opts *backupOptions) ([]string, error) {
	if len(opts.tags) > 0 {
		return opts.tags, nil
	}

	// If no references are specified, discover all tags in the repository
	return registry.Tags(ctx, repo)
}

func parseArtifactsToBackup(artifactRefs string) (repository string, tags []string, err error) {
	// Validate input
	if artifactRefs == "" {
		return "", nil, fmt.Errorf("empty reference")
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
		return "", nil, fmt.Errorf("invalid repository %q: %w", repoParts, err)
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
