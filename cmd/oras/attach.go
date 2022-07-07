package main

import (
	"context"
	"fmt"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/input"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/content"
)

type attachOptions struct {
	option.Common
	option.Remote
	option.Pusher

	targetRef    string
	artifactType string
	fileRefs     []string
}

func attachCmd() *cobra.Command {
	var opts attachOptions
	cmd := &cobra.Command{
		Use:   "attach name[:tag|@digest] file[:type] [file...]",
		Short: "Attach files to an existed manifest",
		Long: `Attach files to an existed manifest

Example - Attach file 'hi.txt' with type 'sig/example' to manifest 'hello:test' in registry 'localhost:5000'
  oras attach localhost:5000/hello:test hi.txt --artifact-type sig/example
`,
		Args: cobra.MinimumNArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			opts.fileRefs = args[1:]
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

	// Packing
	dst, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}
	subject, err := dst.Resolve(ctx, dst.Reference.Reference)
	if err != nil {
		return err
	}
	packOpts := oras.PackArtifactOptions{
		Subject: OCIToArtifact(subject),
	}
	var refs []content.FileReference
	for _, ref := range opts.fileRefs {
		refs = append(refs, content.NewFileReference(input.ParseFileReference(ref, "")))
	}
	ociDescs, err := content.LoadFiles(ctx, store, nil, refs, opts.Verbose)
	if err != nil {
		return err
	}
	orasDescs := make([]artifactspec.Descriptor, len(ociDescs))
	for i := range ociDescs {
		orasDescs[i] = *OCIToArtifact(ociDescs[i])
	}
	manifestDesc, err := oras.PackArtifact(ctx, store, opts.artifactType, orasDescs, packOpts)
	if err != nil {
		return err
	}

	// Prepare Push
	copyOptions := oras.DefaultCopyOptions
	copyOptions.PreCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		name, ok := desc.Annotations[ocispec.AnnotationTitle]
		if !ok {
			if !opts.Verbose {
				return nil
			}
			name = desc.MediaType
		}
		return display.Print("Uploading", display.ShortDigest(desc), name)
	}
	copyOptions.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
		return display.Print("Exists   ", display.ShortDigest(desc), desc.Annotations[ocispec.AnnotationTitle])
	}

	// Push
	err = oras.CopyGraph(ctx, store, dst, manifestDesc, copyOptions.CopyGraphOptions)
	if err != nil {
		return err
	}

	fmt.Println("Files attached to", opts.targetRef)
	fmt.Println("Digest:", manifestDesc.Digest)

	// Export manifest
	return opts.ExportManifest(ctx, manifestDesc, store)
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
