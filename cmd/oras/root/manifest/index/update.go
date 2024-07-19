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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/command"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

type updateOptions struct {
	option.Common
	option.Target

	addArguments    []string
	removeArguments []string
	addTargets      []option.Target
	removeTargets   []option.Target
}

func updateCmd() *cobra.Command {
	var opts updateOptions
	cmd := &cobra.Command{
		Use:   "update",
		Short: "add or remove manifests from an image index",
		Long:  `TBD`,
		Args:  cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.RawReference = args[0]
			repo, _, _ := strings.Cut(opts.RawReference, ":")

			// parse the add manifest arguments
			opts.addTargets = make([]option.Target, len(opts.addArguments))
			if err := parseTargetsFromStrings(cmd, opts.addArguments, opts.addTargets, repo, opts.Remote); err != nil {
				return err
			}

			// parse the remove manifest arguments
			opts.removeTargets = make([]option.Target, len(opts.removeArguments))
			if err := parseTargetsFromStrings(cmd, opts.removeArguments, opts.removeTargets, repo, opts.Remote); err != nil {
				return err
			}

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
	indexTarget, err := opts.NewTarget(opts.Common, logger)
	if err != nil {
		return err
	}
	index, err := fetchIndex(ctx, indexTarget, opts.Reference)
	if err != nil {
		return err
	}
	manifests, err := addManifests(ctx, opts.Common, logger, index.Manifests, opts.addTargets)
	if err != nil {
		return err
	}
	manifests, err = removeManifests(ctx, opts.Common, logger, manifests, opts.removeTargets)
	if err != nil {
		return err
	}
	newDesc, reader := packIndex(&index, manifests)
	return pushIndex(ctx, indexTarget, newDesc, opts.Reference, reader)
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

func addManifests(ctx context.Context, common option.Common, logger logrus.FieldLogger, manifests []ocispec.Descriptor, targets []option.Target) ([]ocispec.Descriptor, error) {
	for _, addTarget := range targets {
		target, err := addTarget.NewReadonlyTarget(ctx, common, logger)
		if err != nil {
			return []ocispec.Descriptor{}, err
		}
		desc, err := oras.Resolve(ctx, target, addTarget.Reference, oras.DefaultResolveOptions)
		if err != nil {
			return []ocispec.Descriptor{}, fmt.Errorf("failed to resolve %s: %w", addTarget.Reference, err)
		}
		desc.Platform, err = getPlatform(ctx, target, addTarget.Reference)
		if err != nil {
			return []ocispec.Descriptor{}, err
		}
		manifests = append(manifests, desc)
	}
	return manifests, nil
}

func removeManifests(ctx context.Context, common option.Common, logger logrus.FieldLogger, manifests []ocispec.Descriptor, targets []option.Target) ([]ocispec.Descriptor, error) {
	set := make(map[digest.Digest]struct{})
	for _, b := range targets {
		target, err := b.NewReadonlyTarget(ctx, common, logger)
		if err != nil {
			return []ocispec.Descriptor{}, err
		}
		desc, err := oras.Resolve(ctx, target, b.Reference, oras.DefaultResolveOptions)
		if err != nil {
			return []ocispec.Descriptor{}, fmt.Errorf("failed to resolve %s: %w", b.Reference, err)
		}
		set[desc.Digest] = struct{}{}
	}
	pointer := len(manifests) - 1
	for i, m := range manifests {
		if _, b := set[m.Digest]; b {
			// swap the to-be-removed manifest to the end of slice
			manifests[i] = manifests[pointer]
			pointer = pointer - 1
		}
	}
	// shrink the slice to remove the manifests
	manifests = manifests[:pointer+1]
	return manifests, nil
}
