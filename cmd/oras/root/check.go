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
	"fmt"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/command"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/descriptor"
)

type checkOptions struct {
	option.Common
	option.Target
}

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
	ctx, logger := command.GetLogger(cmd, &opts.Common)
	target, err := opts.NewTarget(opts.Common, logger)
	if err != nil {
		return err
	}
	desc, err := target.Resolve(ctx, opts.Reference)
	if err != nil {
		return err
	}
	if !descriptor.IsManifest(desc) {
		return fmt.Errorf("the reference %s is not a manifest", opts.Reference)
	}
	if descriptor.IsIndex(desc) {
		return fmt.Errorf("index validation is not yet supported")
	}
	// validate manifest
	manifest, err := checkManifest(ctx, target, desc)
	if err != nil {
		return err
	}
	// validate the blobs
	blobs := []ocispec.Descriptor{manifest.Config}
	blobs = append(blobs, manifest.Layers...)
	if err := checkBlobs(ctx, target, blobs); err != nil {
		return err
	}
	return opts.Printer.Printf("check successful!\n")
}

// checkManifest verifies the mediaType, digest and size against the given
// descriptor and returns the parsed manifest.
func checkManifest(ctx context.Context, target oras.GraphTarget, desc ocispec.Descriptor) (ocispec.Manifest, error) {
	// verify size and digest
	manifestBytes, err := content.FetchAll(ctx, target, desc)
	if err != nil {
		return ocispec.Manifest{}, err
	}
	// parse the fetched manifest
	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return ocispec.Manifest{}, err
	}
	// verify media type
	if manifest.MediaType != desc.MediaType {
		return ocispec.Manifest{}, fmt.Errorf("media type mismatch: %q != %q", manifest.MediaType, desc.MediaType)
	}
	return manifest, nil
}

// checkBlobs verifies the digest and size against the given blob descriptor.
// Should leverage concurrency later.
func checkBlobs(ctx context.Context, target oras.GraphTarget, blobs []ocispec.Descriptor) error {
	for _, blob := range blobs {
		_, err := content.FetchAll(ctx, target, blob)
		if err != nil {
			return err
		}
	}
	return nil
}
