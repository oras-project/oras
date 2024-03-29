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

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/display"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/graph"
	"oras.land/oras/internal/registryutil"
)

type attachOptions struct {
	option.Common
	option.Packer
	option.Target
	option.Format

	artifactType string
	concurrency  int
}

func attachCmd() *cobra.Command {
	var opts attachOptions
	cmd := &cobra.Command{
		Use:   "attach [flags] --artifact-type=<type> <name>{:<tag>|@<digest>} <file>[:<layer_media_type>] [...]",
		Short: "[Preview] Attach files to an existing artifact",
		Long: `[Preview] Attach files to an existing artifact

** This command is in preview and under development. **

Example - Attach file 'hi.txt' with aritifact type 'doc/example' to manifest 'hello:v1' in registry 'localhost:5000':
  oras attach --artifact-type doc/example localhost:5000/hello:v1 hi.txt

Example - Push file "hi.txt" with the custom layer media type 'application/vnd.me.hi':
  oras attach --artifact-type doc/example localhost:5000/hello:v1 hi.txt:application/vnd.me.hi

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
		Args: oerrors.CheckArgs(argument.AtLeast(1), "the destination artifact for attaching."),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.RawReference = args[0]
			opts.FileRefs = args[1:]
			if err := option.Parse(&opts); err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAttach(cmd, &opts)
		},
	}

	cmd.Flags().StringVarP(&opts.artifactType, "artifact-type", "", "", "artifact type")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 5, "concurrency level")
	_ = cmd.MarkFlagRequired("artifact-type")
	opts.EnableDistributionSpecFlag()
	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.Target)
}

func runAttach(cmd *cobra.Command, opts *attachOptions) error {
	ctx, logger := opts.WithContext(cmd.Context())
	annotations, err := opts.LoadManifestAnnotations()
	if err != nil {
		return err
	}
	if len(opts.FileRefs) == 0 && len(annotations[option.AnnotationManifest]) == 0 {
		return &oerrors.Error{
			Err:            errors.New(`neither file nor annotation provided in the command`),
			Usage:          fmt.Sprintf("%s %s", cmd.Parent().CommandPath(), cmd.Use),
			Recommendation: `To attach to an existing artifact, please provide files via argument or annotations via flag "--annotation". Run "oras attach -h" for more options and examples`,
		}
	}
	displayStatus, displayMetadata := display.NewAttachHandler(opts.Template, opts.TTY, cmd.OutOrStdout(), opts.Verbose)

	// prepare manifest
	store, err := file.New("")
	if err != nil {
		return err
	}
	defer store.Close()

	dst, err := opts.NewTarget(opts.Common, logger)
	if err != nil {
		return err
	}
	if err := opts.EnsureReferenceNotEmpty(cmd, true); err != nil {
		return err
	}
	// add both pull and push scope hints for dst repository
	// to save potential push-scope token requests during copy
	ctx = registryutil.WithScopeHint(ctx, dst, auth.ActionPull, auth.ActionPush)
	subject, err := dst.Resolve(ctx, opts.Reference)
	if err != nil {
		return err
	}
	descs, err := loadFiles(ctx, store, annotations, opts.FileRefs, displayStatus)
	if err != nil {
		return err
	}

	// prepare push
	dst, err = displayStatus.TrackTarget(dst)
	if err != nil {
		return err
	}
	graphCopyOptions := oras.DefaultCopyGraphOptions
	graphCopyOptions.Concurrency = opts.concurrency
	displayStatus.UpdateCopyOptions(&graphCopyOptions, store)

	packOpts := oras.PackManifestOptions{
		Subject:             &subject,
		ManifestAnnotations: annotations[option.AnnotationManifest],
		Layers:              descs,
	}
	pack := func() (ocispec.Descriptor, error) {
		return oras.PackManifest(ctx, store, oras.PackManifestVersion1_1, opts.artifactType, packOpts)
	}

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

	// Attach
	root, err := doPush(dst, pack, copy)
	if err != nil {
		return err
	}
	err = displayMetadata.OnCompleted(&opts.Target, root, subject)
	if err != nil {
		return err
	}

	// Export manifest
	return opts.ExportManifest(ctx, store, root)
}
