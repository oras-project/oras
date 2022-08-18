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
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/cache"
	"oras.land/oras/internal/docker"
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

Example - Pull all files:
  oras pull localhost:5000/hello:latest

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
	cmd.Flags().StringVarP(&opts.ManifestConfigRef, "config", "", "", "output manifest config file")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runPull(opts pullOptions) error {
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}
	if repo.Reference.Reference == "" {
		return errors.NewErrInvalidReference(repo.Reference)
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
	copyOptions := oras.DefaultCopyOptions
	configPath, configMediaType := parseFileReference(opts.ManifestConfigRef, "")
	copyOptions.FindSuccessors = func(ctx context.Context, fetcher content.Fetcher, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		successors, err := content.Successors(ctx, fetcher, desc)
		if err != nil {
			return nil, err
		}
		var ret []ocispec.Descriptor
		// For each successor s, save its config to Annotations and skip unnamed content.
		for i, s := range successors {
			// Save the config when:
			// 1) MediaType matches, or
			// 2) MediaType not specified and current node is config.
			// Note: For a manifest, the 0th indexed element is always a
			// manifest config.
			if s.MediaType == configMediaType || (configMediaType == "" && i == 0 && isManifestMediaType(desc.MediaType)) {
				// Add annotation for manifest config
				if s.Annotations == nil {
					s.Annotations = make(map[string]string)
				}
				s.Annotations[ocispec.AnnotationTitle] = configPath
			} else if s.Annotations[ocispec.AnnotationTitle] == "" {
				ss, err := content.Successors(ctx, fetcher, s)
				if err != nil {
					return nil, err
				}
				// Skip s if s is unnamed and has no successors.
				if len(ss) == 0 {
					continue
				}
			}
			ret = append(ret, s)
		}
		return ret, nil
	}

	pulledEmpty := true
	copyOptions.PreCopy = display.StatusPrinter("Downloading", opts.Verbose)
	copyOptions.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		name := desc.Annotations[ocispec.AnnotationTitle]
		if name == "" {
			return nil
		}
		pulledEmpty = false
		return display.Print("Downloaded ", display.ShortDigest(desc), name)
	}

	ctx, _ := opts.SetLoggerLevel()
	var dst = file.New(opts.Output)
	dst.AllowPathTraversalOnWrite = opts.PathTraversal
	dst.DisableOverwrite = opts.KeepOldFiles

	// Copy
	desc, err := oras.Copy(ctx, src, repo.Reference.Reference, dst, repo.Reference.Reference, copyOptions)
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

func isManifestMediaType(mediaType string) bool {
	return mediaType == docker.MediaTypeManifest || mediaType == ocispec.MediaTypeImageManifest
}
