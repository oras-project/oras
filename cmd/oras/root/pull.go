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
	"io"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/fileref"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/graph"
)

type pullOptions struct {
	option.Cache
	option.Common
	option.Platform
	option.Target

	concurrency       int
	KeepOldFiles      bool
	IncludeSubject    bool
	PathTraversal     bool
	Output            string
	ManifestConfigRef string
}

func pullCmd() *cobra.Command {
	var opts pullOptions
	cmd := &cobra.Command{
		Use:   "pull [flags] <name>{:<tag>|@<digest>}",
		Short: "Pull files from a registry or an OCI image layout",
		Long: `Pull files from a registry or an OCI image layout

Example - Pull artifact files from a registry:
  oras pull localhost:5000/hello:v1

Example - Recursively pulling all files from a registry, including subjects of hello:v1:
  oras pull --include-subject localhost:5000/hello:v1

Example - Pull files from an insecure registry:
  oras pull --insecure localhost:5000/hello:v1

Example - Pull files from the HTTP registry:
  oras pull --plain-http localhost:5000/hello:v1

Example - Pull files from a registry with local cache:
  export ORAS_CACHE=~/.oras/cache
  oras pull localhost:5000/hello:v1

Example - Pull files from a registry with certain platform:
  oras pull --platform linux/arm/v5 localhost:5000/hello:v1

Example - Pull all files with concurrency level tuned:
  oras pull --concurrency 6 localhost:5000/hello:v1

Example - Pull artifact files from an OCI image layout folder 'layout-dir':
  oras pull --oci-layout layout-dir:v1

Example - Pull artifact files from an OCI layout archive 'layout.tar':
  oras pull --oci-layout layout.tar:v1
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.RawReference = args[0]
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPull(cmd.Context(), opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.KeepOldFiles, "keep-old-files", "k", false, "do not replace existing files when pulling, treat them as errors")
	cmd.Flags().BoolVarP(&opts.PathTraversal, "allow-path-traversal", "T", false, "allow storing files out of the output directory")
	cmd.Flags().BoolVarP(&opts.IncludeSubject, "include-subject", "", false, "[Preview] recursively pull the subject of artifacts")
	cmd.Flags().StringVarP(&opts.Output, "output", "o", ".", "output directory")
	cmd.Flags().StringVarP(&opts.ManifestConfigRef, "config", "", "", "output manifest config file")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 3, "concurrency level")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runPull(ctx context.Context, opts pullOptions) error {
	ctx, _ = opts.WithContext(ctx)
	// Copy Options
	var printed sync.Map
	copyOptions := oras.DefaultCopyOptions
	copyOptions.Concurrency = opts.concurrency
	var configPath, configMediaType string
	var err error
	if opts.ManifestConfigRef != "" {
		configPath, configMediaType, err = fileref.Parse(opts.ManifestConfigRef, "")
		if err != nil {
			return err
		}
	}
	if opts.Platform.Platform != nil {
		copyOptions.WithTargetPlatform(opts.Platform.Platform)
	}
	var getConfigOnce sync.Once
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
			return rc, display.PrintStatus(target, "Processing ", opts.Verbose)
		})

		nodes, subject, config, err := graph.Successors(ctx, statusFetcher, desc)
		if err != nil {
			return nil, err
		}
		if subject != nil && opts.IncludeSubject {
			nodes = append(nodes, *subject)
		}
		if config != nil {
			getConfigOnce.Do(func() {
				if configPath != "" && (configMediaType == "" || config.MediaType == configMediaType) {
					if config.Annotations == nil {
						config.Annotations = make(map[string]string)
					}
					config.Annotations[ocispec.AnnotationTitle] = configPath
				}
			})
			nodes = append(nodes, *config)
		}

		var ret []ocispec.Descriptor
		for _, s := range nodes {
			if s.Annotations[ocispec.AnnotationTitle] == "" {
				ss, err := content.Successors(ctx, fetcher, s)
				if err != nil {
					return nil, err
				}
				if len(ss) == 0 {
					// skip s if it is unnamed AND has no successors.
					if err := printOnce(&printed, s, "Skipped    ", opts.Verbose); err != nil {
						return nil, err
					}
					continue
				}
			}
			ret = append(ret, s)
		}

		return ret, nil
	}

	target, err := opts.NewReadonlyTarget(ctx, opts.Common)
	if err != nil {
		return err
	}
	if err := opts.EnsureReferenceNotEmpty(); err != nil {
		return err
	}
	src, err := opts.CachedTarget(target)
	if err != nil {
		return err
	}
	dst, err := file.New(opts.Output)
	if err != nil {
		return err
	}
	defer dst.Close()
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
				if err := printOnce(&printed, s, "Restored   ", opts.Verbose); err != nil {
					return err
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
	desc, err := oras.Copy(ctx, src, opts.Reference, dst, opts.Reference, copyOptions)
	if err != nil {
		if errors.Is(err, file.ErrPathTraversalDisallowed) {
			err = fmt.Errorf("%s: %w", "use flag --allow-path-traversal to allow insecurely pulling files outside of working directory", err)
		}
		return err
	}
	if pulledEmpty {
		fmt.Println("Downloaded empty artifact")
	}
	fmt.Println("Pulled", opts.AnnotatedReference())
	fmt.Println("Digest:", desc.Digest)
	return nil
}

// generateContentKey generates a unique key for each content descriptor, using
// its digest and name if applicable.
func generateContentKey(desc ocispec.Descriptor) string {
	return desc.Digest.String() + desc.Annotations[ocispec.AnnotationTitle]
}

func printOnce(printed *sync.Map, s ocispec.Descriptor, msg string, verbose bool) error {
	if _, loaded := printed.LoadOrStore(generateContentKey(s), true); loaded {
		return nil
	}
	return display.PrintStatus(s, msg, verbose)
}
