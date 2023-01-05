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

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

type attachOptions struct {
	option.Common
	option.Packer
	option.Target

	artifactType string
	concurrency  int
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
			opts.FileRefs = args[1:]
			opts.FqdnRef = args[0]
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAttach(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.artifactType, "artifact-type", "", "", "artifact type")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 5, "concurrency level")
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

	// prepare manifest
	store := file.New("")
	defer store.Close()
	store.AllowPathTraversalOnWrite = opts.PathValidationDisabled

	dst, err := opts.NewTarget(opts.Common)
	if err != nil {
		return err
	}
	// sanctity check: should not be tarball
	if opts.Reference == "" {
		return oerrors.NewErrInvalidReferenceStr(opts.FqdnRef)
	}
	subject, err := dst.Resolve(ctx, opts.Reference)
	if err != nil {
		return err
	}
	descs, err := loadFiles(ctx, store, annotations, opts.FileRefs, opts.Verbose)
	if err != nil {
		return err
	}

	// prepare push
	packOpts := oras.PackOptions{
		Subject:             &subject,
		ManifestAnnotations: annotations[option.AnnotationManifest],
	}
	pack := func() (ocispec.Descriptor, error) {
		return oras.Pack(ctx, store, opts.artifactType, descs, packOpts)
	}

	graphCopyOptions := oras.DefaultCopyGraphOptions
	graphCopyOptions.Concurrency = opts.concurrency
	updateDisplayOption(&graphCopyOptions, store, opts.Verbose)
	copy := func(root ocispec.Descriptor) error {
		if root.MediaType == ocispec.MediaTypeArtifactManifest {
			graphCopyOptions.FindSuccessors = func(ctx context.Context, fetcher content.Fetcher, node ocispec.Descriptor) ([]ocispec.Descriptor, error) {
				if content.Equal(node, root) {
					// skip subject
					return descs, nil
				}
				return content.Successors(ctx, fetcher, node)
			}
		}
		return oras.CopyGraph(ctx, store, dst, root, graphCopyOptions)
	}

	root, err := pushArtifact(dst, pack, &packOpts, copy, &graphCopyOptions, opts.Verbose)
	if err != nil {
		return err
	}

	if opts.Type == option.RemoteType {
		// Reassemble a reference with subject digest
		ref := dst.(*remote.Repository).Reference
		ref.Reference = subject.Digest.String()
		opts.FqdnRef = ref.String()
	}
	fmt.Println("Attached to", opts.FullReference())
	fmt.Println("Digest:", root.Digest)

	// Export manifest
	return opts.ExportManifest(ctx, store, root)
}
