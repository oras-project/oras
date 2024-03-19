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
	"os"
	"sync"
	"sync/atomic"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/display/status"
	"oras.land/oras/cmd/oras/internal/display/status/track"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
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
		Args: oerrors.CheckArgs(argument.Exactly(1), "the artifact reference you want to pull"),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.RawReference = args[0]
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPull(cmd, &opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.KeepOldFiles, "keep-old-files", "k", false, "do not replace existing files when pulling, treat them as errors")
	cmd.Flags().BoolVarP(&opts.PathTraversal, "allow-path-traversal", "T", false, "allow storing files out of the output directory")
	cmd.Flags().BoolVarP(&opts.IncludeSubject, "include-subject", "", false, "[Preview] recursively pull the subject of artifacts")
	cmd.Flags().StringVarP(&opts.Output, "output", "o", ".", "output directory")
	cmd.Flags().StringVarP(&opts.ManifestConfigRef, "config", "", "", "output manifest config file")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 3, "concurrency level")
	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.Target)
}

func runPull(cmd *cobra.Command, opts *pullOptions) error {
	ctx, logger := opts.WithContext(cmd.Context())
	// Copy Options
	copyOptions := oras.DefaultCopyOptions
	copyOptions.Concurrency = opts.concurrency
	if opts.Platform.Platform != nil {
		copyOptions.WithTargetPlatform(opts.Platform.Platform)
	}
	target, err := opts.NewReadonlyTarget(ctx, opts.Common, logger)
	if err != nil {
		return err
	}
	if err := opts.EnsureReferenceNotEmpty(cmd, true); err != nil {
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

	desc, layerSkipped, err := doPull(ctx, src, dst, copyOptions, opts)
	if err != nil {
		if errors.Is(err, file.ErrPathTraversalDisallowed) {
			err = fmt.Errorf("%s: %w", "use flag --allow-path-traversal to allow insecurely pulling files outside of working directory", err)
		}
		return err
	}

	// suggest oras copy for pulling layers without annotation
	outWriter := cmd.OutOrStdout()
	if layerSkipped {
		fmt.Fprintf(outWriter, "Skipped pulling layers without file name in %q\n", ocispec.AnnotationTitle)
		fmt.Fprintf(outWriter, "Use 'oras copy %s --to-oci-layout <layout-dir>' to pull all layers.\n", opts.RawReference)
	} else {
		fmt.Fprintln(outWriter, "Pulled", opts.AnnotatedReference())
		fmt.Fprintln(outWriter, "Digest:", desc.Digest)
	}
	return nil
}

func doPull(ctx context.Context, src oras.ReadOnlyTarget, dst oras.GraphTarget, opts oras.CopyOptions, po *pullOptions) (ocispec.Descriptor, bool, error) {
	var configPath, configMediaType string
	var err error
	if po.ManifestConfigRef != "" {
		configPath, configMediaType, err = fileref.Parse(po.ManifestConfigRef, "")
		if err != nil {
			return ocispec.Descriptor{}, false, err
		}
	}

	const (
		promptDownloading = "Downloading"
		promptPulled      = "Pulled     "
		promptProcessing  = "Processing "
		promptSkipped     = "Skipped    "
		promptRestored    = "Restored   "
		promptDownloaded  = "Downloaded "
	)

	dst, err = getTrackedTarget(dst, po.TTY, "Downloading", "Pulled     ")
	if err != nil {
		return ocispec.Descriptor{}, false, err
	}
	if tracked, ok := dst.(track.GraphTarget); ok {
		defer tracked.Close()
	}
	var layerSkipped atomic.Bool
	var printed sync.Map
	var getConfigOnce sync.Once
	opts.FindSuccessors = func(ctx context.Context, fetcher content.Fetcher, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		statusFetcher := content.FetcherFunc(func(ctx context.Context, target ocispec.Descriptor) (fetched io.ReadCloser, fetchErr error) {
			if _, ok := printed.LoadOrStore(generateContentKey(target), true); ok {
				return fetcher.Fetch(ctx, target)
			}
			if po.TTY == nil {
				// none TTY, print status log for first-time fetching
				if err := status.PrintStatus(target, promptDownloading, po.Verbose); err != nil {
					return nil, err
				}
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
			if po.TTY == nil {
				// none TTY, add logs for processing manifest
				return rc, status.PrintStatus(target, promptProcessing, po.Verbose)
			}
			return rc, nil
		})

		nodes, subject, config, err := graph.Successors(ctx, statusFetcher, desc)
		if err != nil {
			return nil, err
		}
		if subject != nil && po.IncludeSubject {
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
			if config.Size != ocispec.DescriptorEmptyJSON.Size || config.Digest != ocispec.DescriptorEmptyJSON.Digest || config.Annotations[ocispec.AnnotationTitle] != "" {
				nodes = append(nodes, *config)
			}
		}

		var ret []ocispec.Descriptor
		for _, s := range nodes {
			if s.Annotations[ocispec.AnnotationTitle] == "" {
				if content.Equal(s, ocispec.DescriptorEmptyJSON) {
					// empty layer
					continue
				}
				if s.Annotations[ocispec.AnnotationTitle] == "" {
					// unnamed layers are skipped
					layerSkipped.Store(true)
				}
				ss, err := content.Successors(ctx, fetcher, s)
				if err != nil {
					return nil, err
				}
				if len(ss) == 0 {
					// skip s if it is unnamed AND has no successors.
					if err := printOnce(&printed, s, promptSkipped, po.Verbose, dst); err != nil {
						return nil, err
					}
					continue
				}
			}
			ret = append(ret, s)
		}

		return ret, nil
	}

	opts.PreCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		if _, ok := printed.LoadOrStore(generateContentKey(desc), true); ok {
			return nil
		}
		if po.TTY == nil {
			// none TTY, print status log for downloading
			return status.PrintStatus(desc, promptDownloading, po.Verbose)
		}
		// TTY
		return nil
	}
	opts.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		// restore named but deduplicated successor nodes
		successors, err := content.Successors(ctx, dst, desc)
		if err != nil {
			return err
		}
		for _, s := range successors {
			if _, ok := s.Annotations[ocispec.AnnotationTitle]; ok {
				if err := printOnce(&printed, s, promptRestored, po.Verbose, dst); err != nil {
					return err
				}
			}
		}
		name, ok := desc.Annotations[ocispec.AnnotationTitle]
		if !ok {
			if !po.Verbose {
				return nil
			}
			name = desc.MediaType
		}
		printed.Store(generateContentKey(desc), true)
		return status.Print(promptDownloaded, status.ShortDigest(desc), name)
	}

	// Copy
	desc, err := oras.Copy(ctx, src, po.Reference, dst, po.Reference, opts)
	return desc, layerSkipped.Load(), err
}

// generateContentKey generates a unique key for each content descriptor, using
// its digest and name if applicable.
func generateContentKey(desc ocispec.Descriptor) string {
	return desc.Digest.String() + desc.Annotations[ocispec.AnnotationTitle]
}

func printOnce(printed *sync.Map, s ocispec.Descriptor, msg string, verbose bool, dst any) error {
	if _, loaded := printed.LoadOrStore(generateContentKey(s), true); loaded {
		return nil
	}
	if tracked, ok := dst.(track.GraphTarget); ok {
		// TTY
		return tracked.Prompt(s, msg)

	}
	// none TTY
	return status.PrintStatus(s, msg, verbose)
}

func getTrackedTarget(gt oras.GraphTarget, tty *os.File, actionPrompt, doneprompt string) (oras.GraphTarget, error) {
	if tty == nil {
		return gt, nil
	}
	tracked, err := track.NewTarget(gt, actionPrompt, doneprompt, tty)
	if err != nil {
		return nil, err
	}
	return tracked, nil
}
