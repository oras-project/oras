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
	"context"
	"errors"
	"fmt"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras/cmd/oras/internal/display"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

type attachOptions struct {
	option.Common
	option.Remote
	option.Packer

	targetRef    string
	artifactType string
	concurrency  int64
}

func attachCmd() *cobra.Command {
	var opts attachOptions
	cmd := &cobra.Command{
		Use:   "attach [flags] --artifact-type=<type> <name>{:<tag>|@<digest>} <file>[:<type>] [...]",
		Short: "[Preview] Attach files to an existing artifact",
		Long: `[Preview] Attach files to an existing artifact

** This command is in preview and under development. **

Example - Attach file 'hi.txt' with type 'doc/example' to manifest 'hello:test' in registry 'localhost:5000'
  oras attach --artifact-type doc/example localhost:5000/hello:test hi.txt

Example - Attach file 'hi.txt' and add annotations from file 'annotation.json'
  oras attach --artifact-type doc/example --annotation-file annotation.json localhost:5000/hello:latest hi.txt

Example - Attach an artifact with manifest annotations
  oras attach --artifact-type doc/example --annotation "key1=val1" --annotation "key2=val2" localhost:5000/hello:latest

Example - Attach file 'hi.txt' and add manifest annotations
  oras attach --artifact-type doc/example --annotation "key=val" localhost:5000/hello:latest hi.txt

Example - Attach file 'hi.txt' and export the pushed manifest to 'manifest.json'
  oras attach --artifact-type doc/example --export-manifest manifest.json localhost:5000/hello:latest hi.txt
`,
		Args: cobra.MinimumNArgs(1),
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
	cmd.Flags().Int64VarP(&opts.concurrency, "concurrency", "", 5, "concurrency level")
	cmd.MarkFlagRequired("artifact-type")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runAttach(opts attachOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	annotations, err := opts.LoadManifestAnnotations()
	if err != nil {
		return err
	}
	if len(opts.FileRefs) == 0 && len(annotations[option.AnnotationManifest]) == 0 {
		return errors.New("no blob or manifest annotation are provided")
	}

	// Prepare manifest
	store := file.New("")
	defer store.Close()
	store.AllowPathTraversalOnWrite = opts.PathValidationDisabled

	dst, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}
	if dst.Reference.Reference == "" {
		return oerrors.NewErrInvalidReference(dst.Reference)
	}
	ociSubject, err := dst.Resolve(ctx, dst.Reference.Reference)
	if err != nil {
		return err
	}
	subject := ociToArtifact(ociSubject)
	ociDescs, err := loadFiles(ctx, store, annotations, opts.FileRefs, opts.Verbose)
	if err != nil {
		return err
	}
	orasDescs := make([]artifactspec.Descriptor, len(ociDescs))
	for i := range ociDescs {
		orasDescs[i] = ociToArtifact(ociDescs[i])
	}
	desc, err := oras.PackArtifact(
		ctx, store, opts.artifactType, orasDescs,
		oras.PackArtifactOptions{
			Subject:             &subject,
			ManifestAnnotations: annotations[option.AnnotationManifest],
		})
	if err != nil {
		return err
	}

	// Prepare Push
	committed := &sync.Map{}
	graphCopyOptions := oras.DefaultCopyGraphOptions
	graphCopyOptions.Concurrency = opts.concurrency
	graphCopyOptions.FindSuccessors = func(ctx context.Context, fetcher content.Fetcher, node ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		if isEqualOCIDescriptor(node, desc) {
			// Skip subject
			return ociDescs, nil
		}
		return content.Successors(ctx, fetcher, node)
	}
	graphCopyOptions.PreCopy = display.StatusPrinter("Uploading", opts.Verbose)
	graphCopyOptions.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		return display.PrintStatus(desc, "Exists   ", opts.Verbose)
	}
	graphCopyOptions.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		if err := display.PrintSuccessorStatus(ctx, desc, "Skipped  ", store, committed, opts.Verbose); err != nil {
			return err
		}
		return display.PrintStatus(desc, "Uploaded ", opts.Verbose)
	}
	// Push
	err = oras.CopyGraph(ctx, store, dst, desc, graphCopyOptions)
	if err != nil {
		return err
	}

	fmt.Println("Attached to", opts.targetRef)
	fmt.Println("Digest:", desc.Digest)

	// Export manifest
	return opts.ExportManifest(ctx, store, desc)
}

func isEqualOCIDescriptor(a, b ocispec.Descriptor) bool {
	return a.Size == b.Size && a.Digest == b.Digest && a.MediaType == b.MediaType
}

// ociToArtifact converts OCI descriptor to artifact descriptor.
func ociToArtifact(desc ocispec.Descriptor) artifactspec.Descriptor {
	return artifactspec.Descriptor{
		MediaType:   desc.MediaType,
		Digest:      desc.Digest,
		Size:        desc.Size,
		URLs:        desc.URLs,
		Annotations: desc.Annotations,
	}
}
