package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	ctxo "github.com/deislabs/oras/pkg/context"
	"github.com/deislabs/oras/pkg/oras"

	"github.com/containerd/containerd/reference"
	"github.com/containerd/containerd/remotes"
	orasdocker "github.com/deislabs/oras/pkg/remotes/docker"
	"github.com/need-being/go-tree"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type discoverOptions struct {
	targetRef    string
	artifactType string
	outputType   string
	verbose      bool

	debug     bool
	configs   []string
	username  string
	password  string
	insecure  bool
	plainHTTP bool
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
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "verbose output")

	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "debug mode")
	cmd.Flags().StringArrayVarP(&opts.configs, "config", "c", nil, "auth config path")
	cmd.Flags().StringVarP(&opts.username, "username", "u", "", "registry username")
	cmd.Flags().StringVarP(&opts.password, "password", "p", "", "registry password")
	cmd.Flags().BoolVarP(&opts.insecure, "insecure", "", false, "allow connections to SSL registry without certs")
	cmd.Flags().BoolVarP(&opts.plainHTTP, "plain-http", "", false, "use plain http and not https")
	return cmd
}

func runDiscover(opts discoverOptions) error {
	ctx := context.Background()
	if opts.debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else if !opts.verbose {
		ctx = ctxo.WithLoggerDiscarded(ctx)
	}

	resolver, dopts := newResolver(opts.username, opts.password, opts.insecure, opts.plainHTTP, opts.configs...)

	discoverer, err := orasdocker.WithDiscover(opts.targetRef, resolver, dopts)
	if err != nil {
		return err
	}

	rootNode := tree.New(opts.targetRef)
	desc, refs, err := getAllReferences(ctx, discoverer, opts.targetRef, opts.artifactType, rootNode, opts.outputType == "tree")
	if err != nil {
		return err
	}

	switch opts.outputType {
	case "tree":
		tree.Print(rootNode)
	case "json":
		printDiscoveredReferencesJSON(desc, *refs)
	default:
		fmt.Println("Discovered", len(*refs), "artifacts referencing", opts.targetRef)
		fmt.Println("Digest:", desc.Digest)

		if len(*refs) != 0 {
			fmt.Println()
			printDiscoveredReferencesTable(*refs, opts.verbose)
		}
	}

	return nil
}

func getAllReferences(ctx context.Context, resolver remotes.Resolver, targetRef string, artifactType string, treeNode *tree.Node, queryGraph bool) (ocispec.Descriptor, *[]artifactspec.Descriptor, error) {
	var results []artifactspec.Descriptor
	spec, err := reference.Parse(targetRef)
	if err != nil {
		return ocispec.Descriptor{}, nil, err
	}

	desc, refs, err := oras.Discover(ctx, resolver, targetRef, artifactType)
	if err != nil {
		if err == reference.ErrObjectRequired {
			return ocispec.Descriptor{}, nil, fmt.Errorf("image reference format is invalid. Please specify <name:tag|name@digest>")
		}
		return ocispec.Descriptor{}, nil, err
	}

	for _, r := range refs {
		branch := treeNode.AddPath(r.ArtifactType, r.Digest)
		if queryGraph {
			nestedRef := fmt.Sprintf("%s@%s", spec.Locator, r.Digest)
			_, refs1, err := getAllReferences(ctx, resolver, nestedRef, "", branch, queryGraph)
			if err != nil {
				return ocispec.Descriptor{}, nil, err
			}
			results = append(results, *refs1...)
		}
	}
	results = append(results, refs...)

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
