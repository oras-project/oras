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
	"fmt"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/option"
)

type attachOptions struct {
	option.Common
	option.Remote
	option.Pusher

	targetRef    string
	artifactType string
}

func attachCmd() *cobra.Command {
	var opts attachOptions
	cmd := &cobra.Command{
		Use:   "attach name[:tag|@digest] file[:type] [file...]",
		Short: "[Preview] Attach files to an existing manifest",
		Long: `[Preview] Attach files to an existing manifest

** This command is in preview and under development. **

Example - Attach file 'hi.txt' with type 'doc/example' to manifest 'hello:test' in registry 'localhost:5000'
  oras attach localhost:5000/hello:test hi.txt --artifact-type doc/example
`,
		Args: cobra.MinimumNArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			opts.FileRefs = args[1:]
			return runAttach(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.artifactType, "artifact-type", "", "", "artifact type")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runAttach(opts attachOptions) error {
	ctx, _ := opts.SetLoggerLevel()

	// Prepare manifest
	store := file.New("")
	defer store.Close()
	dst, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}
	if dst.Reference.Reference == "" {
		return newErrInvalidReference(dst.Reference)
	}
	subject, err := dst.Resolve(ctx, dst.Reference.Reference)
	if err != nil {
		return err
	}
	ociDescs, err := loadFiles(ctx, store, nil, opts.FileRefs, opts.Verbose)
	if err != nil {
		return err
	}
	orasDescs := make([]artifactspec.Descriptor, len(ociDescs))
	for i := range ociDescs {
		orasDescs[i] = *OCIToArtifact(ociDescs[i])
	}
	manifestDesc, err := oras.PackArtifact(
		ctx, store, opts.artifactType, orasDescs,
		oras.PackArtifactOptions{
			Subject: OCIToArtifact(subject),
		})
	if err != nil {
		return err
	}

	// Prepare Push
	graphCopyOptions := oras.DefaultCopyGraphOptions
	graphCopyOptions.PreCopy = display.StatusPrinter("Uploading", opts.Verbose)
	graphCopyOptions.OnCopySkipped = display.StatusPrinter("Existed  ", opts.Verbose)
	graphCopyOptions.PostCopy = display.StatusPrinter("Uploaded ", opts.Verbose)

	// Push
	err = oras.CopyGraph(ctx, store, dst, manifestDesc, graphCopyOptions)
	if err != nil {
		return err
	}

	fmt.Println("Files attached to", opts.targetRef)
	fmt.Println("ORAS Artifact Digest:", manifestDesc.Digest)

	// Export manifest
	return opts.ExportManifest(ctx, store, manifestDesc)
}

// OCIToArtifact converts OCI descriptor to artifact descriptor.
func OCIToArtifact(desc ocispec.Descriptor) *artifactspec.Descriptor {
	return &artifactspec.Descriptor{
		MediaType:   desc.MediaType,
		Digest:      desc.Digest,
		Size:        desc.Size,
		URLs:        desc.URLs,
		Annotations: desc.Annotations,
	}
}
