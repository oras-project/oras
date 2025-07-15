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
)

const (
	outputTypeTar = "tar"
	outputTypeDir = "directory"
)

type backupOptions struct {
	option.Cache
	option.Common
	option.Remote
	option.Terminal

	output           string
	outputType       string // "tar" or "directory"
	includeReferrers bool
	concurrency      int

	rawReference string
	reference    string //contains tag or digest
	// path contains registry and repository for the remote target
	path      string
	extraRefs []string

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
			refs := strings.Split(args[0], ",")
			opts.rawReference = refs[0]
			opts.extraRefs = refs[1:]

			// parse raw reference
			// case: "registry-a.k8s.io/kube-apiserver:,"
			// case: "registry-a.k8s.io/kube-apiserver:,v2"
			ref, err := registry.ParseReference(opts.rawReference)
			if err != nil {
				// TODO: better error message
				return fmt.Errorf("failed to parse reference %q: %w", opts.rawReference, err)
			}
			opts.reference = ref.Reference
			ref.Reference = ""
			opts.path = ref.String()
			// TODO: might need to refactor reference parsing with error handling
			opts.references = append([]string{opts.reference}, opts.extraRefs...)

			if err := option.Parse(cmd, &opts); err != nil {
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
	fmt.Println("rawReference:", opts.rawReference)
	fmt.Println("extraRefs:", opts.extraRefs)
	fmt.Println("reference:", opts.reference)
	fmt.Println("path:", opts.path)
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

	// Prepare remote repo as the source
	src, err := opts.Remote.NewRepository(opts.path, opts.Common, logger)
	if err != nil {
		return err
	}

	// Prepare OCI layout as the destination
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

	dst, err := oci.New(tempDir)
	if err != nil {
		return fmt.Errorf("failed to create OCI store: %w", err)
	}

	if len(opts.references) == 0 {
		return backupRepository(ctx, src, dst, tempDir, opts)
	}
	return backupArtifacts()
}

func backupRepository(ctx context.Context, src *remote.Repository, dst *oci.Store, tempDir string, opts *backupOptions) error {
	// TODO: Implement backup logic for the entire repository
	// TODO: call doCopy()?

	// Start with getting all tags first and then copy
	// Might refactor to do copy while discovering tags later
	tags, err := registry.Tags(ctx, src)
	if err != nil {
		return fmt.Errorf("failed to list tags in repository %s: %w", src.Reference, err)
	}

	copyOpts := oras.CopyOptions{
		CopyGraphOptions: oras.CopyGraphOptions{
			Concurrency: opts.concurrency,
		},
	}
	for _, tag := range tags {
		// TODO: handle concurrency between tags
		// TODO: handle output format
		root, err := oras.Copy(ctx, src, tag, dst, tag, copyOpts)
		if err != nil {
			return fmt.Errorf("failed to copy tag %s: %w", tag, err)
		}
		fmt.Printf("Copied tag: %s, root digest: %s\n", tag, root.Digest)
	}

	return nil
}

func backupArtifacts() error {
	// TODO: Implement backup logic for the artifacts with specific references
	// TODO: call doCopy()?
	return nil
}
