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
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/command"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

// tagRegexp matches valid OCI artifact tags.
// reference: https://github.com/opencontainers/distribution-spec/blob/v1.1.1/spec.md#pulling-manifests
var tagRegexp = regexp.MustCompile(`^[\w][\w.-]{0,127}$`)

type restoreOptions struct {
	option.Common
	option.Remote
	option.Terminal

	// flags
	input            string
	excludeReferrers bool
	dryRun           bool

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
	ctx, logger := command.GetLogger(cmd, &opts.Common)

	// TODO:
	// Create OCI store from the input path
	// Connect to the target registry
	// Resolve the specified tags in the store
	// If no tags are specified, discover all tags in the store
	// Extended copy artifacts from the store to the registry: If exclude-referrers is true, use copy
	// If dry-run is true, do not upload

	// Suppress unused variable warnings during development
	_ = logger

	// prepare the source OCI store
	var srcOCI oras.ReadOnlyGraphTarget
	var err error
	if strings.HasSuffix(opts.input, ".tar") {
		srcOCI, err = oci.NewFromTar(ctx, opts.input)
	} else {
		srcOCI, err = oci.NewWithContext(ctx, opts.input)
	}
	if err != nil {
		return fmt.Errorf("failed to prepare OCI store from input %q: %w", opts.input, err)
	}

	// prepare the target registry
	dstRepo, err := opts.NewRepository(opts.repository, opts.Common, logger)
	if err != nil {
		return fmt.Errorf("failed to prepare target repository %q: %w", opts.repository, err)
	}

	return nil
}

func findTagsToRestore(ctx context.Context, opts *restoreOptions, srcOCI oras.ReadOnlyGraphTarget) ([]string, error) {
	if len(opts.tags) > 0 {
		return opts.tags, nil
	}

	// If no references are specified, discover all tags in the repository
	lister, ok := srcOCI.(registry.TagLister)
	if !ok {
		return nil, fmt.Errorf("the source OCI store does not support tag listing: %T", srcOCI)
	}
	var tags []string
	if err := lister.Tags(ctx, "", func(got []string) error {
		tags = append(tags, got...)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to list tags in repository %q: %w", opts.repository, err)
	}
	return tags, nil
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
