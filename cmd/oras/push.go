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
	"strings"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/option"
)

const (
	tagStaged = "staged"
)

type pushOptions struct {
	option.Common
	option.Remote
	option.Packer

	targetRef         string
	extraRefs         []string
	manifestConfigRef string
	artifactType      string
	concurrency       int64
}

func pushCmd() *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:   "push [flags] <name>[:<tag>[,<tag>][...]] <file>[:<type>] [...]",
		Short: "Push files to remote registry",
		Long: `Push files to remote registry

Example - Push file "hi.txt" with media type "application/vnd.oci.image.layer.v1.tar" (default):
  oras push localhost:5000/hello:latest hi.txt

Example - Push file "hi.txt" and export the pushed manifest to a specified path
  oras push --export-manifest manifest.json localhost:5000/hello:latest hi.txt

Example - Push file "hi.txt" with the custom media type "application/vnd.me.hi":
  oras push localhost:5000/hello:latest hi.txt:application/vnd.me.hi

Example - Push multiple files with different media types:
  oras push localhost:5000/hello:latest hi.txt:application/vnd.me.hi bye.txt:application/vnd.me.bye

Example - Push file "hi.txt" with config type "application/vnd.me.config":
  oras push --artifact-type application/vnd.me.config localhost:5000/hello:latest hi.txt

Example - Push file "hi.txt" with the custom manifest config "config.json" of the custom media type "application/vnd.me.config":
  oras push --config config.json:application/vnd.me.config localhost:5000/hello:latest hi.txt

Example - Push file to the insecure registry:
  oras push --insecure localhost:5000/hello:latest hi.txt

Example - Push file to the HTTP registry:
  oras push --plain-http localhost:5000/hello:latest hi.txt

Example - Push repository with manifest annotations
  oras push --annotation "key=val" localhost:5000/hello:latest

Example - Push repository with manifest annotation file
  oras push --annotation-file annotation.json localhost:5000/hello:latest

Example - Push file "hi.txt" with multiple tags:
  oras push localhost:5000/hello:tag1,tag2,tag3 hi.txt

Example - Push file "hi.txt" with multiple tags and concurrency level tuned:
  oras push --concurrency 6 localhost:5000/hello:tag1,tag2,tag3 hi.txt
`,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.artifactType != "" && opts.manifestConfigRef != "" {
				return errors.New("--artifact-type and --config cannot both be provided")
			}
			return opts.ReadPassword()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			refs := strings.Split(args[0], ",")
			opts.targetRef = refs[0]
			opts.extraRefs = refs[1:]
			opts.FileRefs = args[1:]
			return runPush(opts)
		},
	}
	cmd.Flags().StringVarP(&opts.manifestConfigRef, "config", "", "", "`path` of image config file")
	cmd.Flags().StringVarP(&opts.artifactType, "artifact-type", "", "", "artifact type")
	cmd.Flags().Int64VarP(&opts.concurrency, "concurrency", "", 5, "concurrency level")

	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runPush(opts pushOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	annotations, err := opts.LoadManifestAnnotations()
	if err != nil {
		return err
	}

	// Prepare manifest
	store := file.New("")
	defer store.Close()
	store.AllowPathTraversalOnWrite = opts.PathValidationDisabled

	// Ready to push
	copyOptions := oras.DefaultCopyOptions
	copyOptions.Concurrency = opts.concurrency
	updateDisplayOption(&copyOptions.CopyGraphOptions, store, opts.Verbose)
	desc, err := packManifest(ctx, store, annotations, &opts)
	if err != nil {
		return err
	}

	// Push
	dst, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}
	if tag := dst.Reference.Reference; tag == "" {
		err = oras.CopyGraph(ctx, store, dst, desc, copyOptions.CopyGraphOptions)
	} else {
		desc, err = oras.Copy(ctx, store, tagStaged, dst, tag, copyOptions)
	}
	if err != nil {
		return err
	}

	fmt.Println("Pushed", opts.targetRef)

	if len(opts.extraRefs) != 0 {
		contentBytes, err := content.FetchAll(ctx, store, desc)
		if err != nil {
			return err
		}
		tagBytesNOpts := oras.DefaultTagBytesNOptions
		tagBytesNOpts.Concurrency = opts.concurrency
		if _, err = oras.TagBytesN(ctx, &display.TagManifestStatusPrinter{Repository: dst}, desc.MediaType, contentBytes, opts.extraRefs, tagBytesNOpts); err != nil {
			return err
		}
	}

	fmt.Println("Digest:", desc.Digest)

	// Export manifest
	return opts.ExportManifest(ctx, store, desc)
}

func packManifest(ctx context.Context, store *file.Store, annotations map[string]map[string]string, opts *pushOptions) (ocispec.Descriptor, error) {
	var packOpts oras.PackOptions
	packOpts.ConfigAnnotations = annotations[option.AnnotationConfig]
	packOpts.ManifestAnnotations = annotations[option.AnnotationManifest]

	if opts.manifestConfigRef != "" {
		path, mediatype := parseFileReference(opts.manifestConfigRef, oras.MediaTypeUnknownConfig)
		desc, err := store.Add(ctx, option.AnnotationConfig, mediatype, path)
		if err != nil {
			return ocispec.Descriptor{}, err
		}
		desc.Annotations = packOpts.ConfigAnnotations
		packOpts.ConfigDescriptor = &desc
		packOpts.PackImageManifest = true
	}
	descs, err := loadFiles(ctx, store, annotations, opts.FileRefs, opts.Verbose)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	// pack artifact
	manifestDesc, err := oras.Pack(ctx, store, opts.artifactType, descs, packOpts)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	if err = store.Tag(ctx, manifestDesc, tagStaged); err != nil {
		return ocispec.Descriptor{}, err
	}
	return manifestDesc, nil
}

func updateDisplayOption(opts *oras.CopyGraphOptions, store content.Fetcher, verbose bool) {
	committed := &sync.Map{}
	opts.PreCopy = display.StatusPrinter("Uploading", verbose)
	opts.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		return display.PrintStatus(desc, "Exists   ", verbose)
	}
	opts.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		if err := display.PrintSuccessorStatus(ctx, desc, "Skipped  ", store, committed, verbose); err != nil {
			return err
		}
		return display.PrintStatus(desc, "Uploaded ", verbose)
	}
}
