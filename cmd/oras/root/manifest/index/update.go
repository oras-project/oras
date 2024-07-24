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
	"strings"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/command"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/descriptor"
)

type updateOptions struct {
	option.Common
	option.Target

	extraRefs       []string
	addArguments    []string
	removeArguments []string
}

func updateCmd() *cobra.Command {
	var opts updateOptions
	cmd := &cobra.Command{
		Use:   "update <name>{:<tag>|@<digest>} {--add/--remove/--annotation/--annotation-file} [...]",
		Short: "Update an image index",
		Long: `Update an image index and push to the repository or OCI image layout
		
Example - add one manifest and remove two manifests from an index:
  oras manifest index update localhost:5000/hello:latest --add win64 --remove sha256:xxx --remove arm64
		
Example - update the index referenced by tag1 and tag3, and make tag1 and tag3 point to the
 updated index. If the old index has other tags, they remain pointing to the old index.
  oras manifest index update localhost:5000/hello:tag1,tag3 --remove sha256:xxx --remove sha256:xxx --add s390x
  `,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			refs := strings.Split(args[0], ",")
			opts.RawReference = refs[0]
			opts.extraRefs = refs[1:]
			return option.Parse(cmd, &opts)
			// todo: add EnsureReferenceNotEmpty somewhere
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateIndex(cmd, opts)
		},
	}
	option.ApplyFlags(&opts, cmd.Flags())
	cmd.Flags().StringArrayVarP(&opts.addArguments, "add", "", nil, "manifests to add to the index")
	cmd.Flags().StringArrayVarP(&opts.removeArguments, "remove", "", nil, "manifests to remove from the index")

	return oerrors.Command(cmd, &opts.Target)
}

func updateIndex(cmd *cobra.Command, opts updateOptions) error {
	ctx, logger := command.GetLogger(cmd, &opts.Common)
	target, err := opts.NewTarget(opts.Common, logger)
	if err != nil {
		return err
	}
	index, err := fetchIndex(ctx, target, opts.Reference)
	if err != nil {
		return err
	}
	manifests, err := addManifests(ctx, index.Manifests, target, opts.addArguments)
	if err != nil {
		return err
	}
	manifests, err = removeManifests(ctx, manifests, target, opts.removeArguments)
	if err != nil {
		return err
	}
	desc, content, err := packIndex(&index, manifests)
	if err != nil {
		return err
	}
	return pushIndex(ctx, target, desc, content, opts.Reference, opts.extraRefs)
}

func fetchIndex(ctx context.Context, target oras.ReadOnlyTarget, reference string) (ocispec.Index, error) {
	desc, err := oras.Resolve(ctx, target, reference, oras.DefaultResolveOptions)
	if err != nil {
		return ocispec.Index{}, fmt.Errorf("failed to resolve %s: %w", reference, err)
	}
	contentBytes, err := content.FetchAll(ctx, target, desc)
	if err != nil {
		return ocispec.Index{}, err
	}
	var index ocispec.Index
	if err := json.Unmarshal(contentBytes, &index); err != nil {
		return ocispec.Index{}, err
	}
	return index, nil
}

func addManifests(ctx context.Context, manifests []ocispec.Descriptor, target oras.ReadOnlyTarget, adds []string) ([]ocispec.Descriptor, error) {
	for _, add := range adds {
		desc, content, err := oras.FetchBytes(ctx, target, add, oras.DefaultFetchBytesOptions)
		if err != nil {
			return nil, err
		}
		if descriptor.IsImageManifest(desc) {
			desc.Platform, err = getPlatform(ctx, target, content)
			if err != nil {
				return nil, err
			}
		}
		manifests = append(manifests, desc)
	}
	return manifests, nil
}

func removeManifests(ctx context.Context, manifests []ocispec.Descriptor, target oras.ReadOnlyTarget, removes []string) ([]ocispec.Descriptor, error) {
	set := make(map[digest.Digest]struct{})
	for _, rem := range removes {
		desc, _, err := oras.FetchBytes(ctx, target, rem, oras.DefaultFetchBytesOptions)
		if err != nil {
			return nil, err
		}
		set[desc.Digest] = struct{}{}
	}
	pointer := len(manifests) - 1
	for i, m := range manifests {
		if _, exists := set[m.Digest]; exists {
			// swap the to-be-removed manifest to the end of slice
			manifests[i] = manifests[pointer]
			pointer = pointer - 1
		}
	}
	// shrink the slice to remove the manifests
	manifests = manifests[:pointer+1]
	return manifests, nil
}
