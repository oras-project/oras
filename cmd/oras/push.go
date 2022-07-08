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
	"fmt"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/parse"
	"oras.land/oras/internal/content"
)

const (
	annotationConfig   = "$config"
	annotationManifest = "$manifest"
	tagStaged          = "staged"
)

type pushOptions struct {
	option.Common
	option.Remote
	option.Pusher

	targetRef              string
	fileRefs               []string
	pathValidationDisabled bool
	manifestAnnotations    string
	manifestExport         string
	manifestConfigRef      string
}

func pushCmd() *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:   "push name[:tag|@digest] file[:type] [file...]",
		Short: "Push files to remote registry",
		Long: `Push files to remote registry

Example - Push file "hi.txt" with the "application/vnd.oci.image.layer.v1.tar" media type (default):
  oras push localhost:5000/hello:latest hi.txt

Example - Push file "hi.txt" with the custom "application/vnd.me.hi" media type:
  oras push localhost:5000/hello:latest hi.txt:application/vnd.me.hi

Example - Push multiple files with different media types:
  oras push localhost:5000/hello:latest hi.txt:application/vnd.me.hi bye.txt:application/vnd.me.bye

Example - Push file "hi.txt" with the custom manifest config "config.json" of the custom "application/vnd.me.config" media type:
  oras push --manifest-config config.json:application/vnd.me.config localhost:5000/hello:latest hi.txt

Example - Push file to the insecure registry:
  oras push localhost:5000/hello:latest hi.txt --insecure

Example - Push file to the HTTP registry:
  oras push localhost:5000/hello:latest hi.txt --plain-http
`,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			opts.fileRefs = args[1:]
			return runPush(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.manifestAnnotations, "manifest-annotations", "", "", "manifest annotation file")
	cmd.Flags().BoolVarP(&opts.pathValidationDisabled, "disable-path-validation", "", false, "skip path validation")
	cmd.Flags().StringVarP(&opts.manifestConfigRef, "manifest-config", "", "", "manifest config file")

	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runPush(opts pushOptions) error {
	ctx, _ := opts.SetLoggerLevel()

	// Load annotations
	var annotations map[string]map[string]string
	if opts.manifestAnnotations != "" {
		if err := decodeJSON(opts.manifestAnnotations, &annotations); err != nil {
			return err
		}
	}

	// Prepare manifest
	store := file.New("")
	defer store.Close()
	store.AllowPathTraversalOnWrite = opts.pathValidationDisabled

	// Ready to push
	copyOptions := oras.DefaultCopyOptions
	copyOptions.PreCopy = display.Output("Uploading", opts.Verbose)
	copyOptions.OnCopySkipped = display.Output("Existed  ", opts.Verbose)
	copyOptions.PostCopy = display.Output("Uploaded ", opts.Verbose)
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
	fmt.Println("Digest:", desc.Digest)

	// Export manifest
	return opts.ExportManifest(ctx, desc, store)
}

func decodeJSON(filename string, v interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(v)
}

func packManifest(ctx context.Context, store *file.Store, annotations map[string]map[string]string, opts *pushOptions) (ocispec.Descriptor, error) {
	var packOpts oras.PackOptions
	packOpts.ConfigAnnotations = annotations[annotationConfig]
	packOpts.ManifestAnnotations = annotations[annotationManifest]

	if opts.manifestConfigRef != "" {
		path, mediatype := parse.FileReference(opts.manifestConfigRef, oras.MediaTypeUnknownConfig)
		desc, err := store.Add(ctx, annotationConfig, mediatype, path)
		if err != nil {
			return ocispec.Descriptor{}, err
		}
		desc.Annotations = packOpts.ConfigAnnotations
		packOpts.ConfigDescriptor = &desc
	}
	var refs []content.FileReference
	for _, ref := range opts.fileRefs {
		refs = append(refs, content.NewFileReference(parse.FileReference(ref, "")))
	}
	descs, err := content.LoadFiles(ctx, store, annotations, refs, opts.Verbose)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	if len(descs) == 0 {
		fmt.Println("Uploading empty artifact")
	}
	manifestDesc, err := oras.Pack(ctx, store, descs, packOpts)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	if err := store.Tag(ctx, manifestDesc, tagStaged); err != nil {
		return ocispec.Descriptor{}, err
	}
	return manifestDesc, nil
}
