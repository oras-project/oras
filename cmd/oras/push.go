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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/errcode"
	"oras.land/oras/cmd/oras/internal/display"
	fileref "oras.land/oras/cmd/oras/internal/file"
	"oras.land/oras/cmd/oras/internal/option"
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

	// prepare pack
	packOpts := oras.PackOptions{
		ConfigAnnotations:   annotations[option.AnnotationConfig],
		ManifestAnnotations: annotations[option.AnnotationManifest],
	}
	store := file.New("")
	defer store.Close()
	store.AllowPathTraversalOnWrite = opts.PathValidationDisabled
	if opts.manifestConfigRef != "" {
		path, cfgMediaType := fileref.ParseFileReference(opts.manifestConfigRef, oras.MediaTypeUnknownConfig)
		desc, err := store.Add(ctx, option.AnnotationConfig, cfgMediaType, path)
		if err != nil {
			return err
		}
		desc.Annotations = packOpts.ConfigAnnotations
		packOpts.ConfigDescriptor = &desc
		packOpts.PackImageManifest = true
	}
	descs, err := loadFiles(ctx, store, annotations, opts.FileRefs, opts.Verbose)
	if err != nil {
		return err
	}
	pack := func() (ocispec.Descriptor, error) {
		root, err := oras.Pack(ctx, store, opts.artifactType, descs, packOpts)
		if err != nil {
			return ocispec.Descriptor{}, err
		}
		if err = store.Tag(ctx, root, root.Digest.String()); err != nil {
			return ocispec.Descriptor{}, err
		}
		return root, nil
	}

	// prepare push
	dst, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}
	copyOptions := oras.DefaultCopyOptions
	copyOptions.Concurrency = opts.concurrency
	updateDisplayOption(&copyOptions.CopyGraphOptions, store, opts.Verbose)
	copy := func(root ocispec.Descriptor) error {
		if tag := dst.Reference.Reference; tag == "" {
			err = oras.CopyGraph(ctx, store, dst, root, copyOptions.CopyGraphOptions)
		} else {
			_, err = oras.Copy(ctx, store, root.Digest.String(), dst, tag, copyOptions)
		}
		return err
	}

	// Push
	root, err := pushArtifact(dst, pack, &packOpts, copy, &copyOptions.CopyGraphOptions, opts.Verbose)
	if err != nil {
		return err
	}
	fmt.Println("Pushed", opts.targetRef)

	if len(opts.extraRefs) != 0 {
		contentBytes, err := content.FetchAll(ctx, store, root)
		if err != nil {
			return err
		}
		tagBytesNOpts := oras.DefaultTagBytesNOptions
		tagBytesNOpts.Concurrency = opts.concurrency
		if _, err = oras.TagBytesN(ctx, &display.TagManifestStatusPrinter{Repository: dst}, root.MediaType, contentBytes, opts.extraRefs, tagBytesNOpts); err != nil {
			return err
		}
	}

	fmt.Println("Digest:", root.Digest)

	// Export manifest
	return opts.ExportManifest(ctx, store, root)
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

type packFunc func() (ocispec.Descriptor, error)
type copyFunc func(desc ocispec.Descriptor) error

func pushArtifact(dst *remote.Repository, pack packFunc, packOpts *oras.PackOptions, copy copyFunc, copyOpts *oras.CopyGraphOptions, verbose bool) (ocispec.Descriptor, error) {
	root, err := pack()
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	copyRootAttempted := false
	preCopy := copyOpts.PreCopy
	copyOpts.PreCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		if content.Equal(root, desc) {
			// copyRootAttempted helps track whether the returned error is
			// generated from copying root.
			copyRootAttempted = true
		}
		if preCopy != nil {
			return preCopy(ctx, desc)
		}
		return nil
	}

	// push
	if err = copy(root); err == nil {
		return root, nil
	}

	if !copyRootAttempted || root.MediaType != ocispec.MediaTypeArtifactManifest ||
		!isManifestUnsupported(err) {
		return ocispec.Descriptor{}, err
	}

	if err := display.PrintStatus(root, "Fallback ", verbose); err != nil {
		return ocispec.Descriptor{}, err
	}
	dst.SetReferrersCapability(false)
	packOpts.PackImageManifest = true
	root, err = pack()
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	copyOpts.FindSuccessors = func(ctx context.Context, fetcher content.Fetcher, node ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		if content.Equal(node, root) {
			// skip non-config
			content, err := content.FetchAll(ctx, fetcher, root)
			if err != nil {
				return nil, err
			}
			var manifest ocispec.Manifest
			if err := json.Unmarshal(content, &manifest); err != nil {
				return nil, err
			}
			return []ocispec.Descriptor{manifest.Config}, nil
		}

		// config has no successors
		return nil, nil
	}
	if err = copy(root); err != nil {
		return ocispec.Descriptor{}, err
	}
	return root, nil
}

func isManifestUnsupported(err error) bool {
	var errResp *errcode.ErrorResponse
	if !errors.As(err, &errResp) || errResp.StatusCode != http.StatusBadRequest {
		return false
	}

	var errCode errcode.Error
	if !errors.As(errResp, &errCode) {
		return false
	}

	// As of November 2022, ECR is known to return UNSUPPORTED error when
	// putting an OCI artifact manifest.
	switch errCode.Code {
	case errcode.ErrorCodeManifestInvalid, errcode.ErrorCodeUnsupported:
		return true
	}
	return false
}
