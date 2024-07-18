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
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/command"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/contentutil"
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

			opts.addTargets = make([]option.Target, len(opts.addArguments))
			// parse the add manifest arguments
			for i, a := range opts.addArguments {
				var ref string
				if contentutil.IsDigest(a) {
					ref = fmt.Sprintf("%s@%s", repo, a)
				} else {
					ref = fmt.Sprintf("%s:%s", repo, a)
				}
				opts.addArguments[i] = ref
				m := option.Target{RawReference: ref, Remote: opts.Remote}
				if err := m.Parse(cmd); err != nil {
					return err
				}
				opts.addTargets[i] = m
			}

			opts.removeTargets = make([]option.Target, len(opts.removeArguments))
			// parse the remove manifest arguments
			for i, a := range opts.removeArguments {
				var ref string
				if contentutil.IsDigest(a) {
					ref = fmt.Sprintf("%s@%s", repo, a)
				} else {
					ref = fmt.Sprintf("%s:%s", repo, a)
				}
				opts.removeArguments[i] = ref
				m := option.Target{RawReference: ref, Remote: opts.Remote}
				if err := m.Parse(cmd); err != nil {
					return err
				}
				opts.removeTargets[i] = m
			}

			return option.Parse(cmd, &opts)
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
	// fetch old index
	ctx, logger := command.GetLogger(cmd, &opts.Common)
	oldIndex, err := opts.NewTarget(opts.Common, logger)
	if err != nil {
		return err
	}
	desc, err := oras.Resolve(ctx, oldIndex, opts.Reference, oras.DefaultResolveOptions)
	if err != nil {
		return fmt.Errorf("failed to resolve %s: %w", opts.Reference, err)
	}
	contentBytes, err := content.FetchAll(ctx, oldIndex, desc)
	if err != nil {
		return err
	}
	var index ocispec.Index
	if err := json.Unmarshal(contentBytes, &index); err != nil {
		return err
	}
	manifests := index.Manifests

	// resolve the manifests to add, need to get theirs platform information
	for _, b := range opts.addTargets {
		target, err := b.NewReadonlyTarget(ctx, opts.Common, logger)
		if err != nil {
			return err
		}
		if err := b.EnsureReferenceNotEmpty(cmd, false); err != nil {
			return err
		}
		desc, err := oras.Resolve(ctx, target, b.Reference, oras.DefaultResolveOptions)
		if err != nil {
			return fmt.Errorf("failed to resolve %s: %w", b.Reference, err)
		}
		// detect platform information
		// 1. fetch config descriptor
		configDesc, err := fetchConfigDesc(ctx, target, b.Reference)
		if err != nil {
			return err
		}
		// 2. fetch config content
		contentBytes, err := content.FetchAll(ctx, target, configDesc)
		if err != nil {
			return err
		}
		var config ocispec.Image
		if err := json.Unmarshal(contentBytes, &config); err != nil {
			return err
		}
		// 3. extract platform information
		desc.Platform = &config.Platform

		manifests = append(manifests, desc)
	}

	// resolve the manifests to remove
	set := make(map[digest.Digest]struct{})
	for _, b := range opts.removeTargets {
		target, err := b.NewReadonlyTarget(ctx, opts.Common, logger)
		if err != nil {
			return err
		}
		if err := b.EnsureReferenceNotEmpty(cmd, false); err != nil {
			return err
		}
		desc, err := oras.Resolve(ctx, target, b.Reference, oras.DefaultResolveOptions)
		if err != nil {
			return fmt.Errorf("failed to resolve %s: %w", b.Reference, err)
		}
		set[desc.Digest] = struct{}{}
	}

	pointer := len(manifests) - 1
	for i, m := range manifests {
		if _, b := set[m.Digest]; b {
			// swap
			manifests[i] = manifests[pointer]
			pointer = pointer - 1
		}
	}
	manifests = manifests[:pointer+1]

	// pack the new index
	newIndex := ocispec.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType:    ocispec.MediaTypeImageIndex,
		ArtifactType: index.ArtifactType,
		Manifests:    manifests,
		Subject:      index.Subject,
		Annotations:  index.Annotations,
		// todo: annotations
	}
	content, _ := json.Marshal(newIndex)
	newDesc := ocispec.Descriptor{
		Digest:    digest.FromBytes(content),
		MediaType: ocispec.MediaTypeImageIndex,
		Size:      int64(len(content)),
	}
	reader := bytes.NewReader(content)

	// push the new index
	if err := pushIndex(ctx, oldIndex, newDesc, opts.Reference, reader); err != nil {
		return err
	}
	return nil
}
