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

	"gopkg.in/yaml.v3"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras/cmd/oras/internal/option"

	"github.com/need-being/go-tree"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
)

type discoverOptions struct {
	option.Common
	option.Platform
	option.Target

	artifactType string
	outputType   string
}

func discoverCmd() *cobra.Command {
	var opts discoverOptions
	cmd := &cobra.Command{
		Use:   "discover [flags] <name>{:<tag>|@<digest>}",
		Short: "[Preview] Discover referrers of a manifest in the remote registry",
		Long: `[Preview] Discover referrers of a manifest in the remote registry

** This command is in preview and under development. **

Example - Discover direct referrers of manifest 'hello:v1' in registry 'localhost:5000':
  oras discover localhost:5000/hello:v1

Example - Discover direct referrers via referrers API:
  oras discover --distribution-spec v1.1-referrers-api localhost:5000/hello:v1

Example - Discover direct referrers via tag scheme:
  oras discover --distribution-spec v1.1-referrers-tag localhost:5000/hello:v1

Example - Discover all the referrers of manifest 'hello:v1' in registry 'localhost:5000', displayed in a tree view:
  oras discover -o tree localhost:5000/hello:v1

Example - Discover all the referrers of manifest with annotations, displayed in a tree view:
  oras discover -v -o tree localhost:5000/hello:v1

Example - Discover referrers with type 'test-artifact' of manifest 'hello:v1' in registry 'localhost:5000':
  oras discover --artifact-type test-artifact localhost:5000/hello:v1
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.RawReference = args[0]
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiscover(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.artifactType, "artifact-type", "", "", "artifact type")
	cmd.Flags().StringVarP(&opts.outputType, "output", "o", "table", "format in which to display referrers (table, json, or tree). tree format will also show indirect referrers")
	opts.EnableDistributionSpecFlag()
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runDiscover(opts discoverOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewReadonlyTarget(ctx, opts.Common)
	if err != nil {
		return err
	}
	if err := opts.EnsureReferenceNotEmpty(); err != nil {
		return err
	}

	// discover artifacts
	resolveOpts := oras.DefaultResolveOptions
	resolveOpts.TargetPlatform = opts.Platform.Platform
	desc, err := oras.Resolve(ctx, repo, opts.Reference, resolveOpts)
	if err != nil {
		return err
	}

	if opts.outputType == "tree" {
		root := tree.New(opts.Reference)
		err = fetchAllReferrers(ctx, repo, desc, opts.artifactType, root, &opts)
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

	if n := len(refs); n > 1 {
		fmt.Println("Discovered", n, "artifacts referencing", opts.Reference)
	} else {
		fmt.Println("Discovered", n, "artifact referencing", opts.Reference)
	}
	fmt.Println("Digest:", desc.Digest)
	if len(refs) > 0 {
		fmt.Println()
		printDiscoveredReferrersTable(refs, opts.Verbose)
	}
	return nil
}

func fetchReferrers(ctx context.Context, target oras.ReadOnlyGraphTarget, desc ocispec.Descriptor, artifactType string) ([]ocispec.Descriptor, error) {
	var results []ocispec.Descriptor
	if repo, ok := target.(*remote.Repository); ok {
		err := repo.Referrers(ctx, desc, artifactType, func(referrers []ocispec.Descriptor) error {
			results = append(results, referrers...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		// fill in artifact type and filter
		predecessors, err := target.Predecessors(ctx, desc)
		if err != nil {
			return nil, err
		}
		for _, r := range predecessors {
			var fetched []byte
			if rc, err := target.Fetch(ctx, r); err != nil {
				return nil, err
			} else {
				fetched, err = content.ReadAll(rc, r)
				if err != nil {
					return nil, err
				}
			}
			switch r.MediaType {
			case ocispec.MediaTypeArtifactManifest:
				var artifact ocispec.Artifact
				json.Unmarshal(fetched, &artifact)
				r.ArtifactType = artifact.ArtifactType
			case ocispec.MediaTypeImageManifest:
				var image ocispec.Manifest
				json.Unmarshal(fetched, &image)
				r.ArtifactType = image.Config.MediaType
			}
			if artifactType == "" || artifactType == r.ArtifactType {
				results = append(results, r)
			}
		}
	}
	return results, nil
}

func fetchAllReferrers(ctx context.Context, repo oras.ReadOnlyGraphTarget, desc ocispec.Descriptor, artifactType string, node *tree.Node, opts *discoverOptions) error {
	results, err := fetchReferrers(ctx, repo, desc, artifactType)
	if err != nil {
		return err
	}

	for _, r := range results {
		// Find all indirect referrers
		referrerNode := node.AddPath(r.ArtifactType, r.Digest)
		if opts.Verbose {
			for k, v := range r.Annotations {
				bytes, err := yaml.Marshal(map[string]string{k: v})
				if err != nil {
					return err
				}
				referrerNode.AddPathString(strings.TrimSpace(string(bytes)))
			}
		}
		err := fetchAllReferrers(
			ctx, repo,
			ocispec.Descriptor{
				Digest:    r.Digest,
				Size:      r.Size,
				MediaType: r.MediaType,
			},
			artifactType, referrerNode, opts)
		if err != nil {
			return err
		}
	}
	return nil
}

func printDiscoveredReferrersTable(refs []ocispec.Descriptor, verbose bool) {
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
// image index: https://github.com/opencontainers/image-spec/blob/v1.1.0-rc2/image-index.md#image-index-property-descriptions
func printDiscoveredReferrersJSON(desc ocispec.Descriptor, refs []ocispec.Descriptor) error {
	output := ocispec.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2, // historical value. does not pertain to OCI or docker version
		},
		MediaType: ocispec.MediaTypeImageIndex,
		Manifests: refs,
	}

	return printJSON(output)
}

func printJSON(object interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(object)
}
