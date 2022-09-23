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
	"io"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/descriptor"
)

type pullOptions struct {
	option.Cache
	option.Common
	option.Remote
	option.Platform

	targetRef         string
	KeepOldFiles      bool
	PathTraversal     bool
	Output            string
	ManifestConfigRef string
}

func pullCmd() *cobra.Command {
	var opts pullOptions
	cmd := &cobra.Command{
		Use:   "pull [flags] <name>{:<tag>|@<digest>}",
		Short: "Pull files from remote registry",
		Long: `Pull files from remote registry

Example - Pull all files:
  oras pull localhost:5000/hello:latest

Example - Pull files from the insecure registry:
  oras pull --insecure localhost:5000/hello:latest

Example - Pull files from the HTTP registry:
  oras pull --plain-http localhost:5000/hello:latest

Example - Pull files with local cache:
  export ORAS_CACHE=~/.oras/cache
  oras pull localhost:5000/hello:latest

Example - Pull files with certain platform:
  oras pull --platform linux/arm/v5 localhost:5000/hello:latest
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
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
	var printed sync.Map
	targetPlatform, err := opts.Parse()
	if err != nil {
		return err
	}
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}
	if repo.Reference.Reference == "" {
		return errors.NewErrInvalidReference(repo.Reference)
	}
	src, err := opts.CachedTarget(repo)
	if err != nil {
		return err
	}

	// Copy Options
	copyOptions := oras.DefaultCopyOptions
	configPath, configMediaType := parseFileReference(opts.ManifestConfigRef, "")
	if targetPlatform != nil {
		copyOptions.WithTargetPlatform(targetPlatform)
	}
	copyOptions.FindSuccessors = func(ctx context.Context, fetcher content.Fetcher, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		statusFetcher := content.FetcherFunc(func(ctx context.Context, target ocispec.Descriptor) (fetched io.ReadCloser, fetchErr error) {
			if _, ok := printed.LoadOrStore(generateContentKey(target), true); ok {
				return fetcher.Fetch(ctx, target)
			}

			// print status log for first-time fetching
			if err := display.PrintStatus(target, "Downloading", opts.Verbose); err != nil {
				return nil, err
			}
			rc, err := fetcher.Fetch(ctx, target)
			if err != nil {
				return nil, err
			}
			defer func() {
				if fetchErr != nil {
					rc.Close()
				}
			}()
			if err := display.PrintStatus(target, "Processing ", opts.Verbose); err != nil {
				return nil, err
			}
			return rc, nil
		})
		successors, err := content.Successors(ctx, statusFetcher, desc)
		if err != nil {
			return nil, err
		}
		var ret []ocispec.Descriptor
		// Iterate all the successors to
		// 1) Add name annotation to config if configPath is not empty
		// 2) Skip fetching unnamed leaf nodes
		for i, s := range successors {
			// Save the config when:
			// 1) MediaType matches, or
			// 2) MediaType not specified and current node is config.
			// Note: For a manifest, the 0th indexed element is always a
			// manifest config.
			if (s.MediaType == configMediaType || (configMediaType == "" && i == 0 && descriptor.IsImageManifest(desc))) && configPath != "" {
				// Add annotation for manifest config
				if s.Annotations == nil {
					s.Annotations = make(map[string]string)
				}
				s.Annotations[ocispec.AnnotationTitle] = configPath
			}
			if s.Annotations[ocispec.AnnotationTitle] == "" {
				ss, err := content.Successors(ctx, fetcher, s)
				if err != nil {
					return nil, err
				}
				// Skip s if s is unnamed and has no successors.
				if len(ss) == 0 {
					if _, loaded := printed.LoadOrStore(generateContentKey(s), true); !loaded {
						if err = display.PrintStatus(s, "Skipped    ", opts.Verbose); err != nil {
							return nil, err
						}
					}
					continue
				}
			}
			ret = append(ret, s)
		}
		return ret, nil
	}

	ctx, _ := opts.SetLoggerLevel()
	var dst = file.New(opts.Output)
	dst.AllowPathTraversalOnWrite = opts.PathTraversal
	dst.DisableOverwrite = opts.KeepOldFiles

	pulledEmpty := true
	copyOptions.PreCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		if _, ok := printed.LoadOrStore(generateContentKey(desc), true); ok {
			return nil
		}
		return display.PrintStatus(desc, "Downloading", opts.Verbose)
	}
	copyOptions.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		// restore named but deduplicated successor nodes
		successors, err := content.Successors(ctx, dst, desc)
		if err != nil {
			return err
		}
		for _, s := range successors {
			if _, ok := s.Annotations[ocispec.AnnotationTitle]; ok {
				if _, ok := printed.LoadOrStore(generateContentKey(s), true); !ok {
					if err = display.PrintStatus(s, "Restored   ", opts.Verbose); err != nil {
						return err
					}
				}
			}
		}
		name, ok := desc.Annotations[ocispec.AnnotationTitle]
		if !ok {
			if !opts.Verbose {
				return nil
			}
			name = desc.MediaType
		} else {
			// named content downloaded
			pulledEmpty = false
		}
		printed.Store(generateContentKey(desc), true)
		return display.Print("Downloaded ", display.ShortDigest(desc), name)
	}

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

// generateContentKey generates a unique key for each content descriptor, using
// its digest and name if applicable.
func generateContentKey(desc ocispec.Descriptor) string {
	return desc.Digest.String() + desc.Annotations[ocispec.AnnotationTitle]
}
