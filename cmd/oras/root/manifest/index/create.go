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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/command"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/contentutil"
	"oras.land/oras/internal/descriptor"
)

type createOptions struct {
	option.Common
	option.Target

	sources []option.Target
}

func createCmd() *cobra.Command {
	var opts createOptions
	cmd := &cobra.Command{
		Use:   "create [flags] <name>[:<tag>|@<digest>] [{<tag>|<digest>}...]",
		Short: "create and push an index from provided manifests",
		Long: `create and push an index to a repository or an OCI image layout
Example - create an index from source manifests tagged s1, s2, s3 in the repository
 localhost:5000/hello, and push the index without tagging it :
  oras manifest index create localhost:5000/hello s1 s2 s3
Example - create an index from source manifests tagged s1, s2, s3 in the repository
 localhost:5000/hello, and push the index with tag 'latest' :
  oras manifest index create localhost:5000/hello:latest s1 s2 s3
Example - create an index from source manifests using both tags and digests, 
 and push the index with tag 'latest' :
  oras manifest index create localhost:5000/hello latest s1 sha256:xxx s3
`,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// parse the destination index reference
			opts.RawReference = args[0]
			repo, _, _ := strings.Cut(opts.RawReference, ":")

			// parse the source manifests
			opts.sources = make([]option.Target, len(args)-1)
			if err := parseTargetsFromStrings(cmd, args[1:], opts.sources, repo, opts.Remote); err != nil {
				return err
			}
			return option.Parse(cmd, &opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return createIndex(cmd, opts)
		},
	}

	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.Target)
}

func createIndex(cmd *cobra.Command, opts createOptions) error {
	ctx, logger := command.GetLogger(cmd, &opts.Common)
	dst, err := opts.NewTarget(opts.Common, logger)
	if err != nil {
		return err
	}
	// we assume that the sources and the to be created index are all in the same
	// repository, so no copy is needed
	manifests, err := resolveSourceManifests(cmd, opts, logger)
	if err != nil {
		return err
	}
	desc, reader := packIndex(&ocispec.Index{}, manifests)
	return pushIndex(ctx, dst, desc, opts.Reference, reader)
}

func parseTargetsFromStrings(cmd *cobra.Command, arguments []string, targets []option.Target, repo string, remote option.Remote) error {
	for i, arg := range arguments {
		var ref string
		if contentutil.IsDigest(arg) {
			ref = fmt.Sprintf("%s@%s", repo, arg)
		} else {
			ref = fmt.Sprintf("%s:%s", repo, arg)
		}
		m := option.Target{RawReference: ref, Remote: remote}
		if err := m.Parse(cmd); err != nil {
			return err
		}
		targets[i] = m
	}
	return nil
}

func getPlatform(ctx context.Context, target oras.ReadOnlyTarget, reference string) (*ocispec.Platform, error) {
	// fetch config descriptor
	configDesc, err := fetchConfigDesc(ctx, target, reference)
	if err != nil {
		return &ocispec.Platform{}, err
	}
	// fetch config content
	contentBytes, err := content.FetchAll(ctx, target, configDesc)
	if err != nil {
		return &ocispec.Platform{}, err
	}
	var config ocispec.Image
	if err := json.Unmarshal(contentBytes, &config); err != nil {
		return &ocispec.Platform{}, err
	}
	return &config.Platform, nil
}

func resolveSourceManifests(cmd *cobra.Command, destOpts createOptions, logger logrus.FieldLogger) ([]ocispec.Descriptor, error) {
	var resolved []ocispec.Descriptor
	for _, source := range destOpts.sources {
		var err error
		// prepare sourceTarget target
		sourceTarget, err := source.NewReadonlyTarget(cmd.Context(), destOpts.Common, logger)
		if err != nil {
			return []ocispec.Descriptor{}, err
		}
		if err := source.EnsureReferenceNotEmpty(cmd, false); err != nil {
			return []ocispec.Descriptor{}, err
		}
		var desc ocispec.Descriptor
		desc, err = oras.Resolve(cmd.Context(), sourceTarget, source.Reference, oras.DefaultResolveOptions)
		if err != nil {
			return []ocispec.Descriptor{}, fmt.Errorf("failed to resolve %s: %w", source.Reference, err)
		}
		desc.Platform, err = getPlatform(cmd.Context(), sourceTarget, source.Reference)
		if err != nil {
			return []ocispec.Descriptor{}, err
		}
		resolved = append(resolved, desc)
	}
	return resolved, nil
}

func fetchConfigDesc(ctx context.Context, src oras.ReadOnlyTarget, reference string) (ocispec.Descriptor, error) {
	// fetch manifest descriptor and content
	fetchOpts := oras.DefaultFetchBytesOptions
	manifestDesc, manifestContent, err := oras.FetchBytes(ctx, src, reference, fetchOpts)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	// if this manifest does not have a config
	if !descriptor.IsImageManifest(manifestDesc) {
		return ocispec.Descriptor{}, nil
	}

	// unmarshal manifest content to extract config descriptor
	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestContent, &manifest); err != nil {
		return ocispec.Descriptor{}, err
	}
	return manifest.Config, nil
}

func packIndex(oldIndex *ocispec.Index, manifests []ocispec.Descriptor) (ocispec.Descriptor, io.Reader) {
	index := ocispec.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType:    ocispec.MediaTypeImageIndex,
		ArtifactType: oldIndex.ArtifactType,
		Manifests:    manifests,
		Subject:      oldIndex.Subject,
		Annotations:  oldIndex.Annotations,
	}
	content, _ := json.Marshal(index)
	desc := ocispec.Descriptor{
		Digest:    digest.FromBytes(content),
		MediaType: ocispec.MediaTypeImageIndex,
		Size:      int64(len(content)),
	}
	return desc, bytes.NewReader(content)
}

func pushIndex(ctx context.Context, dst oras.Target, desc ocispec.Descriptor, ref string, content io.Reader) error {
	if refPusher, ok := dst.(registry.ReferencePusher); ok {
		if ref != "" {
			return refPusher.PushReference(ctx, desc, content, ref)
		}
	}
	if err := dst.Push(ctx, desc, content); err != nil {
		w := errors.Unwrap(err)
		if w != errdef.ErrAlreadyExists {
			return err
		}
	}
	if ref == "" {
		fmt.Println("Digest of the pushed index: ", desc.Digest)
		return nil
	}
	return dst.Tag(ctx, desc, ref)
}
