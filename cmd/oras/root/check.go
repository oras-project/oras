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
	"encoding/json"
	"errors"
	"fmt"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/command"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/descriptor"
	"oras.land/oras/internal/docker"
)

type checkOptions struct {
	option.Common
	option.Target
}

// errMismatchedMediaType is returned when the manifest media type is different
// from the descriptor media type.
var errMismatchedMediaType = errors.New("the manifest media type is different from the descriptor media type")

func checkCmd() *cobra.Command {
	var opts checkOptions
	cmd := &cobra.Command{
		Use:   "check [flags] <name>{:<tag>|@<digest>}",
		Short: "TBD",
		Long: `TBD

Example - Check TBD
`,
		Args: oerrors.CheckArgs(argument.Exactly(1), "the artifacts to check"),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.RawReference = args[0]
			return option.Parse(cmd, &opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck(cmd, &opts)
		},
	}
	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.Target)
}

func runCheck(cmd *cobra.Command, opts *checkOptions) error {
	opts.Printer.Verbose = true
	ctx, logger := command.GetLogger(cmd, &opts.Common)
	target, err := opts.NewTarget(opts.Common, logger)
	if err != nil {
		return err
	}
	desc, err := target.Resolve(ctx, opts.Reference)
	if err != nil {
		return err
	}
	errCount, errList := checkGraph(ctx, target, desc, 0, []error{}, opts)
	if errCount > 0 {
		opts.Printer.Printf("\nChecked %s. %d errors found.\n", opts.RawReference, errCount)
		for _, err := range errList {
			opts.Printer.Printf("[Failed]\n")
			opts.Printer.Printf("  - %s\n", err)
		}
		return nil
	}
	return opts.Printer.Printf("\nChecked %s. No errors found.\n", opts.RawReference)
}

// checkGraph
func checkGraph(ctx context.Context, target oras.GraphTarget, root ocispec.Descriptor, errCount int, errList []error, opts *checkOptions) (int, []error) {
	opts.Printer.PrintStatus(root, "Checking          ")
	successors, err := checkNode(ctx, target, root)
	if err != nil {
		opts.Printer.PrintStatus(root, "Checked   [Failed]")
		return errCount + 1, append(errList, err)
	}
	opts.Printer.PrintStatus(root, "Checked   [Pass]  ")
	for _, successor := range successors {
		subgraphErrCount, subGraphErrors := checkGraph(ctx, target, successor, 0, []error{}, opts)
		if len(subGraphErrors) > 0 {
			errCount += subgraphErrCount
			errList = append(errList, subGraphErrors...)
		}
	}
	return errCount, errList
}

// checkNode
func checkNode(ctx context.Context, target oras.GraphTarget, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
	if descriptor.IsManifest(desc) {
		manifestBytes, err := checkManifest(ctx, target, desc)
		if err != nil {
			return nil, err
		}
		return getSuccessors(manifestBytes, desc)
	}
	return nil, checkBlob(ctx, target, desc)
}

// checkManifest verifies the mediaType, digest and size against the given
// descriptor and returns the parsed manifest.
func checkManifest(ctx context.Context, target oras.GraphTarget, desc ocispec.Descriptor) ([]byte, error) {
	// verify size and digest
	manifestBytes, err := content.FetchAll(ctx, target, desc)
	if err != nil {
		switch {
		case errors.Is(err, errdef.ErrNotFound):
			return nil, fmt.Errorf("check failed for manifest %s: manifest not found: %w", desc.Digest, err)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return nil, fmt.Errorf("check failed for manifest %s: invalid manifest size: expect size=%d: %w", desc.Digest, desc.Size, err)
		case errors.Is(err, content.ErrTrailingData):
			return nil, fmt.Errorf("check failed for manifest %s: invalid manifest size: expect size=%d: %w", desc.Digest, desc.Size, err)
		case errors.Is(err, content.ErrMismatchedDigest):
			return nil, fmt.Errorf("check failed for manifest %s: invalid manifest digest: expect digest=%s: %w", desc.Digest, desc.Digest, err)
		default:
			return nil, fmt.Errorf("check failed for manifest %s: %w", desc.Digest, err)
		}
	}
	// parse the fetched manifest
	type mediaType struct {
		MediaType string `json:"mediaType,omitempty"`
	}
	var mt mediaType
	if err := json.Unmarshal(manifestBytes, &mt); err != nil {
		return nil, err
	}
	// verify media type
	if mt.MediaType != desc.MediaType {
		return nil, fmt.Errorf("media type mismatch: %q != %q: %w", mt.MediaType, desc.MediaType, errMismatchedMediaType)
	}
	return manifestBytes, nil
}

// checkBlob verifies the digest and size against the given blob descriptor.
func checkBlob(ctx context.Context, target oras.GraphTarget, desc ocispec.Descriptor) error {
	_, err := content.FetchAll(ctx, target, desc)
	if err != nil {
		switch {
		case errors.Is(err, errdef.ErrNotFound):
			return fmt.Errorf("check failed for blob %s: blob not found: %w", desc.Digest, err)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return fmt.Errorf("check failed for blob %s: invalid blob size: expect size=%d: %w", desc.Digest, desc.Size, err)
		case errors.Is(err, content.ErrTrailingData):
			return fmt.Errorf("check failed for blob %s: invalid blob size: expect size=%d: %w", desc.Digest, desc.Size, err)
		case errors.Is(err, content.ErrMismatchedDigest):
			return fmt.Errorf("check failed for blob %s: invalid blob digest: expect digest=%s: %w", desc.Digest, desc.Digest, err)
		default:
			return fmt.Errorf("check failed for blob %s: %w", desc.Digest, err)
		}
	}
	return nil
}

func getSuccessors(bytes []byte, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
	switch desc.MediaType {
	case docker.MediaTypeManifest:
		// OCI manifest schema can be used to marshal docker manifest
		var manifest ocispec.Manifest
		if err := json.Unmarshal(bytes, &manifest); err != nil {
			return nil, err
		}
		return append([]ocispec.Descriptor{manifest.Config}, manifest.Layers...), nil
	case ocispec.MediaTypeImageManifest:
		var manifest ocispec.Manifest
		if err := json.Unmarshal(bytes, &manifest); err != nil {
			return nil, err
		}
		var nodes []ocispec.Descriptor
		if manifest.Subject != nil {
			nodes = append(nodes, *manifest.Subject)
		}
		nodes = append(nodes, manifest.Config)
		return append(nodes, manifest.Layers...), nil
	case docker.MediaTypeManifestList:
		// OCI manifest index schema can be used to marshal docker manifest list
		var index ocispec.Index
		if err := json.Unmarshal(bytes, &index); err != nil {
			return nil, err
		}
		return index.Manifests, nil
	case ocispec.MediaTypeImageIndex:
		var index ocispec.Index
		if err := json.Unmarshal(bytes, &index); err != nil {
			return nil, err
		}
		var nodes []ocispec.Descriptor
		if index.Subject != nil {
			nodes = append(nodes, *index.Subject)
		}
		return append(nodes, index.Manifests...), nil
	}
	return nil, nil
}
