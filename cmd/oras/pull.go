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
	"fmt"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
	"oras.land/oras/internal/cache"
)

type pullOptions struct {
	option.Common
	option.Remote

	targetRef         string
	cacheRoot         string
	KeepOldFiles      bool
	PathTraversal     bool
	Output            string
	ManifestConfigRef string
}

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
	cmd.Flags().StringVarP(&opts.Output, "output", "o", ".", "output directory")
	cmd.Flags().StringVarP(&opts.ManifestConfigRef, "manifest-config", "", "", "output manifest config file")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runPull(opts pullOptions) error {
	//Cache
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}
	var src oras.Target = repo
	if opts.cacheRoot != "" {
		ociStore, err := oci.New(opts.cacheRoot)
		if err != nil {
			return err
		}
		src = cache.New(repo, ociStore)
	}

	// Copy Options
	copyOptions := oras.CopyOptions{}
	status := output.NewStatus()
	pulledEmpty := true
	copyOptions.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		var name string
		n, m := parseFileRef(opts.ManifestConfigRef, oras.MediaTypeUnknownConfig)
		if m == desc.MediaType {
			name = n
		}
		if name == "" {
			name = desc.Annotations[ocispec.AnnotationTitle]
		}
		if name == "" && opts.Verbose {
			name = desc.MediaType
		}
		if name != "" {
			pulledEmpty = false
			return status.Print("Downloaded", output.ToShort(desc), name)
		}
		return nil
	}

	ctx, _ := opts.SetLoggerLevel()
	var dstStore = file.New(opts.Output)
	dstStore.AllowPathTraversalOnWrite = opts.PathTraversal
	dstStore.DisableOverwrite = opts.KeepOldFiles

	// Copy
	desc, err := oras.Copy(ctx, src, repo.Reference.Reference, dstStore, repo.Reference.Reference, copyOptions)
	if err != nil {
		return err
	}
	if pulledEmpty {
		fmt.Println("Downloaded empty artifact")
	}
	fmt.Println("Pulled", opts.targetRef)
	fmt.Println("Digest:", desc.Digest)
	return nil
}
