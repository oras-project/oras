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
)

type (
	pullOptions struct {
		option.Common
		option.Remote
		option.Pull
		targetRef string
		cacheRoot string
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

	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runPull(opts pullOptions) error {
	ctx, _ := opts.SetLoggerLevel()

	ref, err := registry.ParseReference(opts.targetRef)
	if err != nil {
		return err
	}
	reg, err := opts.NewRegistry(ref.Registry, opts.Common)
	if err != nil {
		return err
	}
	repo, err := reg.Repository(ctx, ref.Repository)
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
	var src, dst oras.Target = repo, dstStore

	if opts.cacheRoot != "" {
		cache, err := oci.New(opts.cacheRoot)
		if err != nil {
			return err
		}
		if _, err = oras.Copy(ctx, src, ref.Reference, cache, ref.Reference); err != nil {
			return err
		}
		src = cache
	}
	desc, err := oras.Copy(ctx, src, ref.Reference, dst, ref.Reference)
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

// TODO: support option 'manifest-config', output manifest config file
