package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras/cmd/oras/internal/option"

	"github.com/containerd/containerd/reference"
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
		Use:   "discover [options] <name:tag|name@digest>",
		Short: "discover artifacts from remote registry",
		Long: `discover artifacts from remote registry

Example - Discover artifacts of type "" linked with the specified reference:
  oras discover --artifact-type application/vnd.cncf.notary.v2 localhost:5000/hello:latest
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return runDiscover(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.artifactType, "artifact-type", "", "", "artifact type")
	cmd.Flags().StringVarP(&opts.outputType, "output", "o", "table", fmt.Sprintf("Format in which to display references (%s, %s, or %s). tree format will show all references including nested", "table", "json", "tree"))
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runDiscover(opts discoverOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}

	// discover artifacts
	root := tree.New(opts.targetRef)
	desc, err := repo.Resolve(ctx, repo.Reference.Reference)
	if err != nil {
		return err
	}
	desc, refs, err := getAllReferences(ctx, repo, desc, opts.artifactType, root, opts.outputType == "tree")
	if err != nil {
		return err
	}

	switch opts.outputType {
	case "tree":
		tree.Print(root)
	case "json":
		printDiscoveredReferencesJSON(desc, *refs)
	default:
		fmt.Println("Discovered", len(*refs), "artifacts referencing", opts.targetRef)
		fmt.Println("Digest:", desc.Digest)

		if len(*refs) != 0 {
			fmt.Println()
			printDiscoveredReferencesTable(*refs, opts.Verbose)
		}
	}
	return nil
}

func getAllReferences(ctx context.Context, repo *remote.Repository, desc ocispec.Descriptor, artifactType string, node *tree.Node, queryGraph bool) (ocispec.Descriptor, *[]artifactspec.Descriptor, error) {
	var results []artifactspec.Descriptor
	err := repo.Referrers(ctx, desc, func(referrers []artifactspec.Descriptor) error {
		for _, r := range referrers {
			if artifactType == "" || artifactType == r.ArtifactType {
				results = append(results, r)
				if queryGraph {
					// Find all referrers
					referrerNode := node.AddPath(r.ArtifactType, r.Digest)
					_, nestedReferrers, err := getAllReferences(
						ctx, repo,
						ocispec.Descriptor{
							Digest:    r.Digest,
							Size:      r.Size,
							MediaType: r.MediaType,
						},
						artifactType, referrerNode, queryGraph)
					if err != nil {
						return err
					}
					results = append(results, *nestedReferrers...)
				}
			}
		}
		return nil
	})
	if err != nil {
		if err == reference.ErrObjectRequired {
			return ocispec.Descriptor{}, nil, fmt.Errorf("image reference format is invalid. Please specify <name:tag|name@digest>")
		}
		return ocispec.Descriptor{}, nil, err
	}
	return desc, &results, nil
}

func printDiscoveredReferencesTable(refs []artifactspec.Descriptor, verbose bool) {
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

func printDiscoveredReferencesJSON(desc ocispec.Descriptor, refs []artifactspec.Descriptor) {
	type reference struct {
		Digest   digest.Digest `json:"digest"`
		Artifact string        `json:"artifactType"`
	}
	output := struct {
		Digest     digest.Digest `json:"digest"`
		References []reference   `json:"references"`
	}{
		Digest:     desc.Digest,
		References: make([]reference, len(refs)),
	}

	for i, ref := range refs {
		output.References[i] = reference{
			Digest:   ref.Digest,
			Artifact: ref.ArtifactType,
		}
	}

	printJSON(output)
}

func printJSON(object interface{}) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	encoder.Encode(object)
}
