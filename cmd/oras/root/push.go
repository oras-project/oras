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
	"strings"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/fileref"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/contentutil"
)

type pushOptions struct {
	option.Common
	option.Packer
	option.ImageSpec
	option.Target

	extraRefs         []string
	manifestConfigRef string
	artifactType      string
	concurrency       int
}

func pushCmd() *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:   "push [flags] <name>[:<tag>[,<tag>][...]] <file>[:<type>] [...]",
		Short: "Push files to a registry or an OCI image layout",
		Long: `Push files to a registry or an OCI image layout

Example - Push file "hi.txt" with media type "application/vnd.oci.image.layer.v1.tar" (default):
  oras push localhost:5000/hello:v1 hi.txt

Example - Push file "hi.txt" and export the pushed manifest to a specified path
  oras push --export-manifest manifest.json localhost:5000/hello:v1 hi.txt

Example - Push file "hi.txt" with the custom media type "application/vnd.me.hi":
  oras push localhost:5000/hello:v1 hi.txt:application/vnd.me.hi

Example - Push multiple files with different media types:
  oras push localhost:5000/hello:v1 hi.txt:application/vnd.me.hi bye.txt:application/vnd.me.bye

Example - Push file "hi.txt" with config type "application/vnd.me.config":
  oras push --artifact-type application/vnd.me.config localhost:5000/hello:v1 hi.txt

Example - Push file "hi.txt" with the custom manifest config "config.json" of the custom media type "application/vnd.me.config":
  oras push --config config.json:application/vnd.me.config localhost:5000/hello:v1 hi.txt

Example - Push file "hi.txt" with specific media type when building the manifest:
  oras push --image-spec v1.1-image localhost:5000/hello:v1 hi.txt    # OCI image
  oras push --image-spec v1.1-artifact localhost:5000/hello:v1 hi.txt # OCI artifact

Example - Push file to the insecure registry:
  oras push --insecure localhost:5000/hello:v1 hi.txt

Example - Push file to the HTTP registry:
  oras push --plain-http localhost:5000/hello:v1 hi.txt

Example - Push repository with manifest annotations
  oras push --annotation "key=val" localhost:5000/hello:v1

Example - Push repository with manifest annotation file
  oras push --annotation-file annotation.json localhost:5000/hello:v1

Example - Push file "hi.txt" with multiple tags:
  oras push localhost:5000/hello:tag1,tag2,tag3 hi.txt

Example - Push file "hi.txt" with multiple tags and concurrency level tuned:
  oras push --concurrency 6 localhost:5000/hello:tag1,tag2,tag3 hi.txt

Example - Push file "hi.txt" into an OCI image layout folder 'layout-dir' with tag 'test':
  oras push --oci-layout layout-dir:test hi.txt
`,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			refs := strings.Split(args[0], ",")
			opts.RawReference = refs[0]
			opts.extraRefs = refs[1:]
			opts.FileRefs = args[1:]
			if opts.manifestConfigRef != "" {
				if opts.artifactType != "" {
					return errors.New("--artifact-type and --config cannot both be provided")
				}
				if opts.ManifestMediaType == ocispec.MediaTypeArtifactManifest {
					return errors.New("cannot build an OCI artifact with manifest config")
				}
			}
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVarP(&opts.manifestConfigRef, "config", "", "", "`path` of image config file")
	cmd.Flags().StringVarP(&opts.artifactType, "artifact-type", "", "", "artifact type")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 5, "concurrency level")

	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runPush(ctx context.Context, opts pushOptions) error {
	ctx, _ = opts.WithContext(ctx)
	annotations, err := opts.LoadManifestAnnotations()
	if err != nil {
		return err
	}

	// prepare pack
	packOpts := oras.PackOptions{
		ConfigAnnotations:   annotations[option.AnnotationConfig],
		ManifestAnnotations: annotations[option.AnnotationManifest],
	}
	store, err := file.New("")
	if err != nil {
		return err
	}
	defer store.Close()
	if opts.manifestConfigRef != "" {
		path, cfgMediaType, err := fileref.Parse(opts.manifestConfigRef, oras.MediaTypeUnknownConfig)
		if err != nil {
			return err
		}
		desc, err := store.Add(ctx, option.AnnotationConfig, cfgMediaType, path)
		if err != nil {
			return err
		}
		desc.Annotations = packOpts.ConfigAnnotations
		packOpts.ConfigDescriptor = &desc
		packOpts.PackImageManifest = true
	}
	if opts.ManifestMediaType == ocispec.MediaTypeImageManifest {
		packOpts.PackImageManifest = true
	}
	descs, err := loadFiles(ctx, store, annotations, opts.FileRefs, opts.Verbose)
	if err != nil {
		return err
	}
	memoryStore := memory.New()
	pack := func() (ocispec.Descriptor, error) {
		root, err := oras.Pack(ctx, memoryStore, opts.artifactType, descs, packOpts)
		if err != nil {
			return ocispec.Descriptor{}, err
		}
		if err = memoryStore.Tag(ctx, root, root.Digest.String()); err != nil {
			return ocispec.Descriptor{}, err
		}
		return root, nil
	}

	// prepare push
	dst, err := opts.NewTarget(opts.Common)
	if err != nil {
		return err
	}
	copyOptions := oras.DefaultCopyOptions
	copyOptions.Concurrency = opts.concurrency
	union := contentutil.MultiReadOnlyTarget(memoryStore, store)
	updateDisplayOption(&copyOptions.CopyGraphOptions, union, opts.Verbose)
	copy := func(root ocispec.Descriptor) error {
		if tag := opts.Reference; tag == "" {
			err = oras.CopyGraph(ctx, union, dst, root, copyOptions.CopyGraphOptions)
		} else {
			_, err = oras.Copy(ctx, union, root.Digest.String(), dst, tag, copyOptions)
		}
		return err
	}

	// Push
	root, err := pushArtifact(dst, pack, copy)
	if err != nil {
		return err
	}
	fmt.Println("Pushed", opts.AnnotatedReference())

	if len(opts.extraRefs) != 0 {
		contentBytes, err := content.FetchAll(ctx, memoryStore, root)
		if err != nil {
			return err
		}
		tagBytesNOpts := oras.DefaultTagBytesNOptions
		tagBytesNOpts.Concurrency = opts.concurrency
		if _, err = oras.TagBytesN(ctx, display.NewTagStatusPrinter(dst), root.MediaType, contentBytes, opts.extraRefs, tagBytesNOpts); err != nil {
			return err
		}
	}

	fmt.Println("Digest:", root.Digest)

	// Export manifest
	return opts.ExportManifest(ctx, memoryStore, root)
}

func updateDisplayOption(opts *oras.CopyGraphOptions, fetcher content.Fetcher, verbose bool) {
	committed := &sync.Map{}
	opts.PreCopy = display.StatusPrinter("Uploading", verbose)
	opts.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		return display.PrintStatus(desc, "Exists   ", verbose)
	}
	opts.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		if err := display.PrintSuccessorStatus(ctx, desc, "Skipped  ", fetcher, committed, verbose); err != nil {
			return err
		}
		return display.PrintStatus(desc, "Uploaded ", verbose)
	}
}

type packFunc func() (ocispec.Descriptor, error)
type copyFunc func(desc ocispec.Descriptor) error

func pushArtifact(dst oras.Target, pack packFunc, copy copyFunc) (ocispec.Descriptor, error) {
	root, err := pack()
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	// push
	if err = copy(root); err != nil {
		return ocispec.Descriptor{}, err
	}
	return root, nil
}
