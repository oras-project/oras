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
	"errors"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/display/status"
	"oras.land/oras/cmd/oras/internal/display/status/track"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/fileref"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/contentutil"
	"oras.land/oras/internal/registryutil"
)

type pushOptions struct {
	option.Common
	option.Packer
	option.ImageSpec
	option.Target
	option.Format

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

Example - Push file "hi.txt" and export the pushed manifest to a specified path:
  oras push --export-manifest manifest.json localhost:5000/hello:v1 hi.txt

Example - Push file "hi.txt" with the custom media type "application/vnd.me.hi":
  oras push localhost:5000/hello:v1 hi.txt:application/vnd.me.hi

Example - Push multiple files with different media types:
  oras push localhost:5000/hello:v1 hi.txt:application/vnd.me.hi bye.txt:application/vnd.me.bye

Example - Push file "hi.txt" with artifact type "application/vnd.example+type":
  oras push --artifact-type application/vnd.example+type localhost:5000/hello:v1 hi.txt

Example - Push file "hi.txt" with config type "application/vnd.me.config":
  oras push --image-spec v1.0 --artifact-type application/vnd.me.config localhost:5000/hello:v1 hi.txt

Example - Push file "hi.txt" with the custom manifest config "config.json" of the custom media type "application/vnd.me.config":
  oras push --config config.json:application/vnd.me.config localhost:5000/hello:v1 hi.txt

Example - Push file to the insecure registry:
  oras push --insecure localhost:5000/hello:v1 hi.txt

Example - Push file to the HTTP registry:
  oras push --plain-http localhost:5000/hello:v1 hi.txt

Example - Push repository with manifest annotations:
  oras push --annotation "key=val" localhost:5000/hello:v1

Example - Push repository with manifest annotation file:
  oras push --annotation-file annotation.json localhost:5000/hello:v1

Example - Push file "hi.txt" with multiple tags:
  oras push localhost:5000/hello:tag1,tag2,tag3 hi.txt

Example - Push file "hi.txt" with multiple tags and concurrency level tuned:
  oras push --concurrency 6 localhost:5000/hello:tag1,tag2,tag3 hi.txt

Example - Push file "hi.txt" into an OCI image layout folder 'layout-dir' with tag 'test':
  oras push --oci-layout layout-dir:test hi.txt
`,
		Args: oerrors.CheckArgs(argument.AtLeast(1), "the destination for pushing"),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			refs := strings.Split(args[0], ",")
			opts.RawReference = refs[0]
			opts.extraRefs = refs[1:]
			opts.FileRefs = args[1:]
			if err := option.Parse(&opts); err != nil {
				return err
			}
			switch opts.PackVersion {
			case oras.PackManifestVersion1_0:
				if opts.manifestConfigRef != "" && opts.artifactType != "" {
					return errors.New("--artifact-type and --config cannot both be provided for 1.0 OCI image")
				}
			case oras.PackManifestVersion1_1:
				if opts.manifestConfigRef == "" && opts.artifactType == "" {
					opts.artifactType = oras.MediaTypeUnknownArtifact
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(cmd, &opts)
		},
	}
	cmd.Flags().StringVarP(&opts.manifestConfigRef, "config", "", "", "`path` of image config file")
	cmd.Flags().StringVarP(&opts.artifactType, "artifact-type", "", "", "artifact type")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 5, "concurrency level")

	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.Target)
}

func runPush(cmd *cobra.Command, opts *pushOptions) error {
	ctx, logger := opts.WithContext(cmd.Context())
	annotations, err := opts.LoadManifestAnnotations()
	if err != nil {
		return err
	}
	displayStatus, displayMetadata := display.NewPushHandler(opts.Template, opts.TTY, cmd.OutOrStdout(), opts.Verbose)

	// prepare pack
	packOpts := oras.PackManifestOptions{
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
		desc, err := addFile(ctx, store, option.AnnotationConfig, cfgMediaType, path)
		if err != nil {
			return err
		}
		desc.Annotations = packOpts.ConfigAnnotations
		packOpts.ConfigDescriptor = &desc
	}
	descs, err := loadFiles(ctx, store, annotations, opts.FileRefs, displayStatus)
	if err != nil {
		return err
	}
	packOpts.Layers = descs
	memoryStore := memory.New()
	pack := func() (ocispec.Descriptor, error) {
		root, err := oras.PackManifest(ctx, memoryStore, opts.PackVersion, opts.artifactType, packOpts)
		if err != nil {
			return ocispec.Descriptor{}, err
		}
		if err = memoryStore.Tag(ctx, root, root.Digest.String()); err != nil {
			return ocispec.Descriptor{}, err
		}
		return root, nil
	}

	// prepare push
	dst, err := opts.NewTarget(opts.Common, logger)
	if err != nil {
		return err
	}
	dst, err = displayStatus.TrackTarget(dst)
	if err != nil {
		return err
	}
	copyOptions := oras.DefaultCopyOptions
	copyOptions.Concurrency = opts.concurrency
	union := contentutil.MultiReadOnlyTarget(memoryStore, store)
	displayStatus.UpdateCopyOptions(&copyOptions.CopyGraphOptions, union)
	copy := func(root ocispec.Descriptor) error {
		// add both pull and push scope hints for dst repository
		// to save potential push-scope token requests during copy
		ctx = registryutil.WithScopeHint(ctx, dst, auth.ActionPull, auth.ActionPush)

		if tag := opts.Reference; tag == "" {
			err = oras.CopyGraph(ctx, union, dst, root, copyOptions.CopyGraphOptions)
		} else {
			_, err = oras.Copy(ctx, union, root.Digest.String(), dst, tag, copyOptions)
		}
		return err
	}

	// Push
	root, err := doPush(dst, pack, copy)
	if err != nil {
		return err
	}
	err = displayMetadata.OnCopied(&opts.Target)
	if err != nil {
		return err
	}

	if len(opts.extraRefs) != 0 {
		taggable := dst
		if tracked, ok := dst.(track.GraphTarget); ok {
			taggable = tracked.Inner()
		}
		contentBytes, err := content.FetchAll(ctx, memoryStore, root)
		if err != nil {
			return err
		}
		tagBytesNOpts := oras.DefaultTagBytesNOptions
		tagBytesNOpts.Concurrency = opts.concurrency
		if _, err = oras.TagBytesN(ctx, status.NewTagStatusPrinter(taggable), root.MediaType, contentBytes, opts.extraRefs, tagBytesNOpts); err != nil {
			return err
		}
	}

	err = displayMetadata.OnCompleted(root)
	if err != nil {
		return err
	}

	// Export manifest
	return opts.ExportManifest(ctx, memoryStore, root)
}

func doPush(dst oras.Target, pack packFunc, copy copyFunc) (ocispec.Descriptor, error) {
	if tracked, ok := dst.(track.GraphTarget); ok {
		defer tracked.Close()
	}
	// Push
	return pushArtifact(dst, pack, copy)
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
