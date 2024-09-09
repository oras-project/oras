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

package index

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/command"
	"oras.land/oras/cmd/oras/internal/display/status"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
	"oras.land/oras/internal/contentutil"
	"oras.land/oras/internal/descriptor"
)

type updateOptions struct {
	option.Common
	option.Target

	addArguments    []string
	mergeArguments  []string
	removeArguments []string
	tags            []string
}

func updateCmd() *cobra.Command {
	var opts updateOptions
	cmd := &cobra.Command{
		Use:   "update <name>{:<tag>|@<digest>} {--add/--merge/--remove} {<tag>|<digest>} [...]",
		Short: "[Experimental] Update and push an image index",
		Long: `[Experimental] Update and push an image index. All manifests should be in the same repository
		
Example - remove a manifest and add two manifests from an index tagged 'v1'. The tag will point to the updated index:
  oras manifest index update localhost:5000/hello:v1 --add linux-amd64 --add linux-arm64 --remove sha256:99e4703fbf30916f549cd6bfa9cdbab614b5392fbe64fdee971359a77073cdf9

Example - update an index by specifying its digest:
  oras manifest index update localhost:5000/hello@sha256:99e4703fbf30916f549cd6bfa9cdbab614b5392fbe64fdee971359a77073cdf9 --add linux-amd64 --remove linux-arm64

Example - merge manifests from the index 'v1' to the index 'v2':
  oras manifest index update localhost:5000/hello:v2 --merge v1

Example - update an index and tag the updated index as 'v2.1.0' and 'latest':
  oras manifest index update localhost:5000/hello@sha256:99e4703fbf30916f549cd6bfa9cdbab614b5392fbe64fdee971359a77073cdf9 --add linux-amd64 --tag "v2.1.0" --tag "latest"
  `,
		Args: oerrors.CheckArgs(argument.AtLeast(1), "the destination index to update"),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.RawReference = args[0]
			return option.Parse(cmd, &opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateIndex(cmd, opts)
		},
	}
	option.ApplyFlags(&opts, cmd.Flags())
	cmd.Flags().StringArrayVarP(&opts.addArguments, "add", "", nil, "add manifests to the index")
	cmd.Flags().StringArrayVarP(&opts.mergeArguments, "merge", "", nil, "merge the manifests of another index")
	cmd.Flags().StringArrayVarP(&opts.removeArguments, "remove", "", nil, "manifests to remove from the index")
	cmd.Flags().StringArrayVarP(&opts.tags, "tag", "", nil, "tags for the updated index")
	return oerrors.Command(cmd, &opts.Target)
}

func updateIndex(cmd *cobra.Command, opts updateOptions) error {
	// if no update flag is used, do nothing
	if !cmd.Flags().Changed("add") && !cmd.Flags().Changed("remove") && !cmd.Flags().Changed("merge") {
		opts.Println("No update flag is used. There's nothing to update.")
		return nil
	}
	ctx, logger := command.GetLogger(cmd, &opts.Common)
	target, err := opts.NewTarget(opts.Common, logger)
	if err != nil {
		return err
	}
	if err := opts.EnsureReferenceNotEmpty(cmd, true); err != nil {
		return err
	}
	index, err := fetchIndex(ctx, target, opts)
	if err != nil {
		return err
	}
	manifests, err := addManifests(ctx, index.Manifests, target, opts)
	if err != nil {
		return err
	}
	manifests, err = mergeIndexes(ctx, manifests, target, opts)
	if err != nil {
		return err
	}
	manifests, err = removeManifests(ctx, manifests, target, opts)
	if err != nil {
		return err
	}

	// media type may be converted to "application/vnd.oci.image.index.v1+json"
	updatedIndex := ocispec.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType:    ocispec.MediaTypeImageIndex,
		ArtifactType: index.ArtifactType,
		Manifests:    manifests,
		Subject:      index.Subject,
		Annotations:  index.Annotations,
	}
	indexBytes, _ := json.Marshal(updatedIndex)
	desc := content.NewDescriptorFromBytes(updatedIndex.MediaType, indexBytes)

	printUpdateStatus(status.IndexPromptUpdated, string(desc.Digest), "", opts.Printer)
	return pushIndex(ctx, target, desc, indexBytes, opts.Reference, opts.tags, opts.AnnotatedReference(), opts.Printer)
}

func fetchIndex(ctx context.Context, target oras.ReadOnlyTarget, opts updateOptions) (ocispec.Index, error) {
	printUpdateStatus(status.IndexPromptFetching, opts.Reference, "", opts.Printer)
	desc, content, err := oras.FetchBytes(ctx, target, opts.Reference, oras.DefaultFetchBytesOptions)
	if err != nil {
		return ocispec.Index{}, fmt.Errorf("could not find the index %s: %w", opts.Reference, err)
	}
	if !descriptor.IsIndex(desc) {
		return ocispec.Index{}, fmt.Errorf("%s is not an index", opts.Reference)
	}
	printUpdateStatus(status.IndexPromptFetched, opts.Reference, string(desc.Digest), opts.Printer)
	var index ocispec.Index
	if err := json.Unmarshal(content, &index); err != nil {
		return ocispec.Index{}, err
	}
	return index, nil
}

func addManifests(ctx context.Context, manifests []ocispec.Descriptor, target oras.ReadOnlyTarget, opts updateOptions) ([]ocispec.Descriptor, error) {
	for _, manifestRef := range opts.addArguments {
		printUpdateStatus(status.IndexPromptFetching, manifestRef, "", opts.Printer)
		desc, content, err := oras.FetchBytes(ctx, target, manifestRef, oras.DefaultFetchBytesOptions)
		if err != nil {
			return nil, fmt.Errorf("could not find the manifest %s: %w", manifestRef, err)
		}
		if !descriptor.IsManifest(desc) {
			return nil, fmt.Errorf("%s is not a manifest", manifestRef)
		}
		printUpdateStatus(status.IndexPromptFetched, manifestRef, string(desc.Digest), opts.Printer)
		if descriptor.IsImageManifest(desc) {
			desc.Platform, err = getPlatform(ctx, target, content)
			if err != nil {
				return nil, err
			}
		}
		manifests = append(manifests, desc)
		printUpdateStatus(status.IndexPromptAdded, manifestRef, string(desc.Digest), opts.Printer)
	}
	return manifests, nil
}

func mergeIndexes(ctx context.Context, manifests []ocispec.Descriptor, target oras.ReadOnlyTarget, opts updateOptions) ([]ocispec.Descriptor, error) {
	for _, indexRef := range opts.mergeArguments {
		printUpdateStatus(status.IndexPromptFetching, indexRef, "", opts.Printer)
		desc, content, err := oras.FetchBytes(ctx, target, indexRef, oras.DefaultFetchBytesOptions)
		if err != nil {
			return nil, fmt.Errorf("could not find the index %s: %w", indexRef, err)
		}
		if !descriptor.IsIndex(desc) {
			return nil, fmt.Errorf("%s is not an index", indexRef)
		}
		printUpdateStatus(status.IndexPromptFetched, indexRef, string(desc.Digest), opts.Printer)
		var index ocispec.Index
		if err := json.Unmarshal(content, &index); err != nil {
			return nil, err
		}
		manifests = append(manifests, index.Manifests...)
		printUpdateStatus(status.IndexPromptMerged, indexRef, string(desc.Digest), opts.Printer)
	}
	return manifests, nil
}

func removeManifests(ctx context.Context, manifests []ocispec.Descriptor, target oras.ReadOnlyTarget, opts updateOptions) ([]ocispec.Descriptor, error) {
	digestCounter := make(map[digest.Digest]int)
	for _, manifestRef := range opts.removeArguments {
		printUpdateStatus(status.IndexPromptResolving, manifestRef, "", opts.Printer)
		desc, err := oras.Resolve(ctx, target, manifestRef, oras.DefaultResolveOptions)
		if err != nil {
			return nil, fmt.Errorf("could not resolve the manifest %s: %w", manifestRef, err)
		}
		if !descriptor.IsManifest(desc) {
			return nil, fmt.Errorf("%s is not a manifest", manifestRef)
		}
		printUpdateStatus(status.IndexPromptResolved, manifestRef, string(desc.Digest), opts.Printer)
		digestCounter[desc.Digest] = digestCounter[desc.Digest] + 1
	}
	pointer := len(manifests) - 1
	for i := len(manifests) - 1; i >= 0; i-- {
		if _, exists := digestCounter[manifests[i].Digest]; exists {
			val := manifests[i]
			// move the item to the end of the slice
			for j := i; j < pointer; j++ {
				manifests[j] = manifests[j+1]
			}
			manifests[pointer] = val
			pointer = pointer - 1
			printUpdateStatus(status.IndexPromptRemoved, string(val.Digest), "", opts.Printer)
			digestCounter[val.Digest] = digestCounter[val.Digest] - 1
			if digestCounter[val.Digest] == 0 {
				delete(digestCounter, val.Digest)
			}
		}
	}
	// shrink the slice to remove the manifests
	manifests = manifests[:pointer+1]
	for key := range digestCounter {
		return nil, fmt.Errorf("%s does not exist in the index %s", key, opts.Reference)
	}
	return manifests, nil
}

func printUpdateStatus(verb string, reference string, resolvedDigest string, printer *output.Printer) {
	if resolvedDigest == "" || contentutil.IsDigest(reference) {
		printer.Println(verb, reference)
	} else {
		printer.Println(verb, resolvedDigest, reference)
	}
}
