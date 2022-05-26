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
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/status"
)

type (
	pullOptions struct {
		option.Common
		option.Remote

		targetRef         string
		cacheRoot         string
		KeepOldFiles      bool
		PathTraversal     bool
		Output            string
		ManifestConfigRef string
	}
)

func pullCmd() *cobra.Command {
	var opts pullOptions
	cmd := &cobra.Command{
		Use:   "pull <name:tag|name@digest>",
		Short: "Pull files from remote registry",
		Long: `Pull files from remote registry

Example - Pull only files with the "application/vnd.oci.image.layer.v1.tar" media type (default):
  oras pull localhost:5000/hello:latest

Example - Pull only files with the custom "application/vnd.me.hi" media type:
  oras pull localhost:5000/hello:latest -t application/vnd.me.hi

Example - Pull all files, any media type:
  oras pull localhost:5000/hello:latest -a

Example - Pull files from the insecure registry:
  oras pull localhost:5000/hello:latest --insecure

Example - Pull files from the HTTP registry:
  oras pull localhost:5000/hello:latest --plain-http

Example - Pull files with local cache:
  export ORAS_CACHE=~/.oras/cache
  oras pull localhost:5000/hello:latest
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.cacheRoot = os.Getenv("ORAS_CACHE")
			return opts.ReadPassword()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return runPull(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.KeepOldFiles, "keep-old-files", "k", false, "do not replace existing files when pulling, treat them as errors")
	cmd.Flags().BoolVarP(&opts.PathTraversal, "allow-path-traversal", "T", false, "allow storing files out of the output directory")
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "", "output directory")
	cmd.Flags().StringVarP(&opts.ManifestConfigRef, "manifest-config", "", "", "output manifest config file")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runPull(opts pullOptions) error {
	ctx, _ := opts.SetLoggerLevel()

	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}

	dir := opts.Output
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	var dstStore = file.New(dir)
	dstStore.AllowPathTraversalOnWrite = opts.PathTraversal
	dstStore.DisableOverwrite = opts.KeepOldFiles

	var mco *status.ManifestConfigOption
	if opts.ManifestConfigRef != "" {
		name, media := parseFileRef(opts.ManifestConfigRef, oras.MediaTypeUnknownConfig)
		mco = &status.ManifestConfigOption{
			Name:      name,
			MediaType: media,
		}
	}
	var src, dst oras.Target = repo, dstStore
	tracker := status.NewPullTracker(dst, mco)
	ref, err := registry.ParseReference(opts.targetRef)
	if err != nil {
		return err
	}

	if opts.cacheRoot != "" {
		cache, err := oci.New(opts.cacheRoot)
		if err != nil {
			return err
		}
		if _, err = oras.Copy(ctx, src, ref.Reference, cache, ref.ReferenceOrDefault()); err != nil {
			return err
		}
		src = cache
	}
	desc, err := oras.Copy(ctx, src, ref.Reference, tracker, ref.Reference)
	if err != nil {
		return err
	}
	artifacts, err := content.DownEdges(ctx, src, desc)
	if err != nil {
		return err
	}

	if len(artifacts) == 0 {
		fmt.Println("Downloaded empty artifact")
	}
	fmt.Println("Pulled", opts.targetRef)
	fmt.Println("Digest:", desc.Digest)

	return nil
}
