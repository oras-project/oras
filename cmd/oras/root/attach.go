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
	"errors"
	"fmt"
	"os"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	oerr "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/graph"
)

type attachOptions struct {
	option.Common
	option.Packer
	option.ImageSpec
	option.Target
	option.Referrers

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

Example - Attach file 'hi.txt' with type 'doc/example' to manifest 'hello:v1' in registry 'localhost:5000':
  oras attach --artifact-type doc/example localhost:5000/hello:v1 hi.txt

Example - Attach file "hi.txt" with specific media type when building the manifest:
  oras attach --artifact-type doc/example --image-spec v1.1-image localhost:5000/hello:v1 hi.txt    # OCI image
  oras attach --artifact-type doc/example --image-spec v1.1-artifact localhost:5000/hello:v1 hi.txt # OCI artifact

Example - Attach file "hi.txt" using a specific method for the Referrers API:
  oras attach --artifact-type doc/example --distribution-spec v1.1-referrers-api localhost:5000/hello:v1 hi.txt # via API
  oras attach --artifact-type doc/example --distribution-spec v1.1-referrers-tag localhost:5000/hello:v1 hi.txt # via tag scheme

Example - Attach file 'hi.txt' and add annotations from file 'annotation.json':
  oras attach --artifact-type doc/example --annotation-file annotation.json localhost:5000/hello:v1 hi.txt

Example - Attach an artifact with manifest annotations:
  oras attach --artifact-type doc/example --annotation "key1=val1" --annotation "key2=val2" localhost:5000/hello:v1

Example - Attach file 'hi.txt' and add manifest annotations:
  oras attach --artifact-type doc/example --annotation "key=val" localhost:5000/hello:v1 hi.txt

Example - Attach file 'hi.txt' and export the pushed manifest to 'manifest.json':
  oras attach --artifact-type doc/example --export-manifest manifest.json localhost:5000/hello:v1 hi.txt

Example - Attach file to the manifest tagged 'v1' in an OCI image layout folder 'layout-dir':
  oras attach --oci-layout --artifact-type doc/example layout-dir:v1 hi.txt
  `,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.RawReference = args[0]
			opts.FileRefs = args[1:]
			if err := option.Parse(&opts); err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAttach(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.artifactType, "artifact-type", "", "", "artifact type")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 5, "concurrency level")
	_ = cmd.MarkFlagRequired("artifact-type")
	opts.EnableDistributionSpecFlag()
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runAttach(ctx context.Context, opts attachOptions) error {
	ctx, logger := opts.WithContext(ctx)
	annotations, err := opts.LoadManifestAnnotations()
	if err != nil {
		return err
	}
	if len(opts.FileRefs) == 0 && len(annotations[option.AnnotationManifest]) == 0 {
		return errors.New("no blob or manifest annotation are provided")
	}

	// prepare manifest
	store, err := file.New("")
	if err != nil {
		return err
	}
	defer store.Close()

	dst, err := opts.NewTarget(opts.Common)
	if err != nil {
		return err
	}
	if err := opts.EnsureReferenceNotEmpty(); err != nil {
		return err
	}
	opts.SetReferrersGC(dst, logger)

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
		PackImageManifest:   opts.ManifestMediaType == ocispec.MediaTypeImageManifest,
	}
	pack := func() (ocispec.Descriptor, error) {
		return oras.Pack(ctx, store, opts.artifactType, descs, packOpts)
	}

	graphCopyOptions := oras.DefaultCopyGraphOptions
	graphCopyOptions.Concurrency = opts.concurrency
	updateDisplayOption(&graphCopyOptions, store, opts.Verbose)
	copy := func(root ocispec.Descriptor) error {
		graphCopyOptions.FindSuccessors = func(ctx context.Context, fetcher content.Fetcher, node ocispec.Descriptor) ([]ocispec.Descriptor, error) {
			if content.Equal(node, root) {
				// skip duplicated Resolve on subject
				successors, _, config, err := graph.Successors(ctx, fetcher, node)
				if err != nil {
					return nil, err
				}
				if config != nil {
					successors = append(successors, *config)
				}
				return successors, nil
			}
			return content.Successors(ctx, fetcher, node)
		}
		return oras.CopyGraph(ctx, store, dst, root, graphCopyOptions)
	}

	root, err := pushArtifact(dst, pack, copy)
	if err != nil {
		if oerr.IsReferrersIndexDelete(err) {
			fmt.Fprintln(os.Stderr, "attached successfully but failed to remove the outdated referrers index, please use `--skip-delete-referrers` if you want to skip the deletion")
		}
		return err
	}

	digest := subject.Digest.String()
	if !strings.HasSuffix(opts.RawReference, digest) {
		opts.RawReference = fmt.Sprintf("%s@%s", opts.Path, subject.Digest)
	}
	fmt.Println("Attached to", opts.AnnotatedReference())
	fmt.Println("Digest:", root.Digest)

	// Export manifest
	return opts.ExportManifest(ctx, store, root)
}
