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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"

	"github.com/need-being/go-tree"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
	"github.com/spf13/cobra"
)

type discoverOptions struct {
	option.Common
	option.Remote

	targetRef    string
	artifactType string
	outputType   string
}

func discoverCmd() *cobra.Command {
	var opts discoverOptions
	cmd := &cobra.Command{
		Use:   "discover [options] <name>{:<tag>|@<digest>}",
		Short: "[Preview] Discover referrers of a manifest in the remote registry",
		Long: `[Preview] Discover referrers of a manifest in the remote registry

** This command is in preview and under development. **

Example - Discover direct referrers of manifest 'hello:latest' in registry 'localhost:5000':
  oras discover localhost:5000/hello

Example - Discover all the referrers of manifest 'hello:latest' in registry 'localhost:5000' in a tree view:
  oras discover -o tree localhost:5000/hello

Example - Discover referrers with type 'test-artifact' of manifest 'hello:latest' in registry 'localhost:5000':
  oras discover --artifact-type test-artifact localhost:5000/hello
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return runDiscover(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.artifactType, "artifact-type", "", "", "artifact type")
	cmd.Flags().StringVarP(&opts.outputType, "output", "o", "table", "format in which to display referrers (table, json, or tree). tree format will also show indirect referrers")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runDiscover(opts discoverOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}
	if repo.Reference.Reference == "" {
		return errors.NewErrInvalidReference(repo.Reference)
	}

	// discover artifacts
	desc, err := repo.Resolve(ctx, repo.Reference.Reference)
	if err != nil {
		return err
	}

	if opts.outputType == "tree" {
		root := tree.New(repo.Reference.String())
		err = fetchAllReferrers(ctx, repo, desc, opts.artifactType, root)
		if err != nil {
			return err
		}
		return tree.Print(root)
	}

	refs, err := fetchReferrers(ctx, repo, desc, opts.artifactType)
	if err != nil {
		return err
	}
	if opts.outputType == "json" {
		return printDiscoveredReferrersJSON(desc, refs)
	}

	fmt.Println("Discovered", len(refs), "artifacts referencing", repo.Reference)
	fmt.Println("Digest:", desc.Digest)
	if len(refs) > 0 {
		fmt.Println()
		printDiscoveredReferrersTable(refs, opts.Verbose)
	}
	return nil
}

func fetchReferrers(ctx context.Context, repo *remote.Repository, desc ocispec.Descriptor, artifactType string) ([]artifactspec.Descriptor, error) {
	var results []artifactspec.Descriptor
	err := repo.Referrers(ctx, desc, artifactType, func(referrers []artifactspec.Descriptor) error {
		results = append(results, referrers...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func fetchAllReferrers(ctx context.Context, repo *remote.Repository, desc ocispec.Descriptor, artifactType string, node *tree.Node) error {
	results, err := fetchReferrers(ctx, repo, desc, artifactType)
	if err != nil {
		return err
	}

	for _, r := range results {
		// Find all indirect referrers
		referrerNode := node.AddPath(r.ArtifactType, r.Digest)
		err := fetchAllReferrers(
			ctx, repo,
			ocispec.Descriptor{
				Digest:    r.Digest,
				Size:      r.Size,
				MediaType: r.MediaType,
			},
			artifactType, referrerNode)
		if err != nil {
			return err
		}
	}
	return nil
}

func printDiscoveredReferrersTable(refs []artifactspec.Descriptor, verbose bool) {
	typeNameTitle := "Artifact Type"
	typeNameLength := len(typeNameTitle)
	for _, ref := range refs {
		if length := len(ref.ArtifactType); length > typeNameLength {
			typeNameLength = length
		}
	}

	print := func(key string, value interface{}) {
		fmt.Println(key, strings.Repeat(" ", typeNameLength-len(key)+1), value)
	}

	print(typeNameTitle, "Digest")
	for _, ref := range refs {
		print(ref.ArtifactType, ref.Digest)
		if verbose {
			printJSON(ref)
		}
	}
}

// printDiscoveredReferrersJSON prints referrer list in JSON equivalent to the
// API result: https://github.com/oras-project/artifacts-spec/blob/v1.0.0-rc.1/manifest-referrers-api.md#artifact-referrers-api-results
func printDiscoveredReferrersJSON(desc ocispec.Descriptor, refs []artifactspec.Descriptor) error {
	type referrerDesc struct {
		Digest    digest.Digest `json:"digest"`
		MediaType string        `json:"mediaType"`
		Artifact  string        `json:"artifactType"`
		Size      int64         `json:"size"`
	}
	output := struct {
		Referrers []referrerDesc `json:"referrers"`
	}{
		Referrers: make([]referrerDesc, len(refs)),
	}

	for i, ref := range refs {
		output.Referrers[i] = referrerDesc{
			Digest:    ref.Digest,
			Artifact:  ref.ArtifactType,
			Size:      ref.Size,
			MediaType: ref.MediaType,
		}
	}

	return printJSON(output)
}

func printJSON(object interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(object)
}
