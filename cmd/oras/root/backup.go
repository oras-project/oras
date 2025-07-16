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
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/command"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/io"
)

const (
	outputTypeTar = "tar"
	outputTypeDir = "directory"
)

// tagRegexp checks the tag name.
// The docker and OCI spec have the same regular expression.
//
// Reference: https://github.com/opencontainers/distribution-spec/blob/v1.1.1/spec.md#pulling-manifests
var tagRegexp = regexp.MustCompile(`^[\w][\w.-]{0,127}$`)

type backupOptions struct {
	option.Cache
	option.Common
	option.Remote
	option.Terminal

	output           string
	outputType       string // "tar" or "directory"
	includeReferrers bool
	concurrency      int

	repository string
	references []string
}

func backupCmd() *cobra.Command {
	var opts backupOptions
	cmd := &cobra.Command{
		Use:   "backup [flags] --output <path> <registry>/<repository>[:<ref1>[,<ref2>...]]",
		Short: "Back up artifacts from a registry to a local directory or tar file",
		Long: `Back up artifacts from a registry to a local directory or tar file

Example - Back up artifact with referrers from a registry to a tar file:
  oras backup --output backup.tar --include-referrers registry-a.k8s.io/kube-apiserver

Example - Back up specific tagged artifacts with referrers:
  oras backup --output backup.tar --include-referrers registry-a.k8s.io/kube-apiserver:v1,v2

Example - Back up artifact from an insecure registry:
  oras backup --output backup.tar --insecure localhost:5000/hello:v1

Example - Back up artifact from the HTTP registry:
  oras backup --output backup.tar --plain-http localhost:5000/hello:v1

Example - Back up artifact with local cache:
  export ORAS_CACHE=~/.oras/cache
  oras backup --output backup.tar registry.com/myrepo:v1

Example - Back up with concurrency level tuned:
  oras backup --output backup.tar --concurrency 6 registry.com/myrepo:v1
`,
		Args: oerrors.CheckArgs(argument.Exactly(1), "the artifact reference you want to back up"),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := option.Parse(cmd, &opts); err != nil {
				return err
			}

			// parse repo and references
			var err error
			opts.repository, opts.references, err = parseArtifactRefs(args[0])
			if err != nil {
				return err
			}

			// parse output type
			if strings.HasSuffix(opts.output, ".tar") {
				opts.outputType = outputTypeTar
			} else {
				opts.outputType = outputTypeDir
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

	// debugging
	fmt.Println("******OPTIONS******")
	fmt.Println("path:", opts.repository)
	fmt.Println("references:", opts.references)
	fmt.Println("output:", opts.output)
	fmt.Println("outputType:", opts.outputType)
	fmt.Println("includeReferrers:", opts.includeReferrers)
	fmt.Println("******END OF OPTIONS******")

	// TODO:
	// Overall, copy the artifacts from remote to OCI layout, and create a tar file if output type is tar
	// If no references is specified: discover all tags in the repository and copy them
	// If references are specified: copy the specified reference and extra refs
	// If includeReferrers is true: do extended copy (questions: handle multi-arch?)

	// TODO: might need to refactor output type handling here
	var dstRoot string
	switch opts.outputType {
	case outputTypeDir:
		dstRoot = opts.output
	case outputTypeTar:
		tempDir, err := os.MkdirTemp("", "oras-backup-*")
		if err != nil {
			// TODO: better error message?
			return fmt.Errorf("failed to create temporary directory: %w", err)
		}
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				logger.Warnf("failed to remove temporary directory %s: %v", tempDir, err)
			}
		}()
		dstRoot = tempDir
	default:
		// this should not happen
		return fmt.Errorf("unsupported output type: %s", opts.outputType)
	}

	// Prepare remote repo as the source
	src, err := opts.Remote.NewRepository(opts.repository, opts.Common, logger)
	if err != nil {
		return err
	}
	// Prepare OCI layout as the destination
	dst, err := oci.New(dstRoot)
	if err != nil {
		return fmt.Errorf("failed to create OCI store: %w", err)
	}

	tags := referencesToBackup(ctx, src, opts)
	if len(tags) == 0 {
		// TODO: better error message
		return fmt.Errorf("no references to back up, please specify at least one reference")
	}

	// Do the backup
	copyOpts := oras.CopyOptions{
		CopyGraphOptions: oras.CopyGraphOptions{
			Concurrency: opts.concurrency,
		},
	}
	extendedCopyOpts := oras.ExtendedCopyOptions{
		ExtendedCopyGraphOptions: oras.ExtendedCopyGraphOptions{
			CopyGraphOptions: oras.CopyGraphOptions{
				Concurrency: opts.concurrency,
			},
		},
	}
	for _, tag := range tags {
		// TODO: handle concurrency between refs
		// TODO: handle output format
		fmt.Println("Found ref:", tag)
		if opts.includeReferrers {
			root, err := oras.ExtendedCopy(ctx, src, tag, dst, tag, extendedCopyOpts)
			if err != nil {
				return fmt.Errorf("failed to extended copy ref %s: %w", tag, err)
			}
			fmt.Printf("Extended copied ref: %s, root digest: %s\n", tag, root.Digest)
		} else {
			root, err := oras.Copy(ctx, src, tag, dst, tag, copyOpts)
			if err != nil {
				return fmt.Errorf("failed to copy ref %s: %w", tag, err)
			}
			fmt.Printf("Copied ref: %s, root digest: %s\n", tag, root.Digest)
		}
	}

	if opts.outputType != outputTypeTar {
		return nil
	}

	// TODO: remove ingest dir from dstRoot

	// get a tarFile writer by creating the tarFile or replace if existing
	// TODO: refactor for better structure?
	tempTar, err := os.CreateTemp("", "oras-backup-*.tar")
	if err != nil {
		return fmt.Errorf("failed to create output tar file %s: %w", opts.output, err)
	}
	if err := io.TarDirectory(ctx, tempTar, dstRoot); err != nil {
		return fmt.Errorf("failed to write tar file %s: %w", opts.output, err)
	}
	if err := tempTar.Close(); err != nil {
		logger.Warnf("failed to close tar file %s: %v", opts.output, err)
	}
	if err := os.Rename(tempTar.Name(), opts.output); err != nil {
		return fmt.Errorf("failed to rename tar file %s to %s: %w", tempTar.Name(), opts.output, err)
	}

	fmt.Println("Successfully backed up artifacts to", opts.output)
	return nil
}

func referencesToBackup(ctx context.Context, repo *remote.Repository, opts *backupOptions) []string {
	if len(opts.references) > 0 {
		// TODO: handle reference, e.g., tag or digest
		return opts.references
	}

	// If no references are specified, discover all tags in the repository
	tags, err := registry.Tags(ctx, repo)
	if err != nil {
		return nil
	}
	return tags
}

func parseArtifactRefs(artifactRefs string) (repository string, tags []string, err error) {
	// TODO: more tests
	// Reject digest references
	if len(artifactRefs) == 0 {
		return "", nil, fmt.Errorf("invalid reference format: empty reference")
	}
	if strings.Contains(artifactRefs, "@") {
		return "", nil, fmt.Errorf("digest references are not supported: %q", artifactRefs)
	}

	refs := strings.Split(artifactRefs, ",")
	artifactRef := refs[0]
	extraRefs := refs[1:]

	// Validate the main reference
	parsedRef, err := registry.ParseReference(artifactRef)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse reference %q: %w", artifactRef, err)
	}

	// Process references
	if parsedRef.Reference == "" {
		tags = extraRefs[:]
	} else {
		tags = append([]string{parsedRef.Reference}, extraRefs...)
	}

	// Strip the reference part to get the repository
	parsedRef.Reference = ""
	repository = parsedRef.String()

	for _, tag := range tags {
		if !tagRegexp.MatchString(tag) {
			return "", nil, fmt.Errorf("invalid tag %q in reference %q", tag, artifactRefs)
		}
	}

	// TODO: validate each reference against tagRegex?
	return repository, tags, nil
}
