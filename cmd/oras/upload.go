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
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/option"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

const (
	directory = "output"
)

type uploadOptions struct {
	option.Common
	To option.Remote
}

func uploadCmd() *cobra.Command {
	var opts uploadOptions
	cmd := &cobra.Command{
		Use:     "upload [flags] <source>{:<tag>|@<digest>} [<source>{:<tag>|@<digest>}][...]]",
		Aliases: []string{"upload"},
		Short:   "[Preview] upload multiple artifacts",
		Long: `[Preview] upload multiple artifacts

** This command is in preview and under development. **

Example - uplad artifacts
  oras upload localhost:5000/owner/this:v1 localhost:6000/owner/that:v1
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runupload(opts, args[0])
		},
	}
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runupload(opts uploadOptions, destination string) error {
	ctx, _ := opts.SetLoggerLevel()

	var uniqueImages []registry.Reference
	err := filepath.Walk(directory,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			ociPath, layout := strings.CutSuffix(path, "/oci-layout")
			if layout == true {
				ociPath, _ = strings.CutPrefix(ociPath, directory+"/")
				lastIdx := strings.LastIndex(ociPath, "/")
				reference := registry.Reference{
					Repository: ociPath[:lastIdx],
					Reference:  ociPath[lastIdx+1:],
				}
				uniqueImages = append(uniqueImages, reference)
			}
			return nil
		})
	if err != nil {
		return fmt.Errorf("problem traversing directory %s: %v", directory, err)
	}

	// Prepare upload options
	var src option.ReadOnlyGraphTagFinderTarget
	committed := &sync.Map{}
	extendedCopyOptions := oras.DefaultExtendedCopyOptions
	extendedCopyOptions.Concurrency = concurrency
	extendedCopyOptions.PreCopy = display.StatusPrinter("uploading", opts.Verbose)
	extendedCopyOptions.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		if err := display.PrintSuccessorStatus(ctx, desc, "Skipped", src, committed, opts.Verbose); err != nil {
			return err
		}
		return display.PrintStatus(desc, "uploaded ", opts.Verbose)
	}
	extendedCopyOptions.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		return display.PrintStatus(desc, "Exists ", opts.Verbose)
	}
	uploadOptions := oras.CopyOptions{
		CopyGraphOptions: extendedCopyOptions.CopyGraphOptions,
	}

	for _, fileReference := range uniqueImages {
		// Prepare source
		var from option.Target
		from.Type = option.TargetTypeOCILayout
		from.Path = path.Join(directory, fileReference.Repository, fileReference.Reference)
		from.Reference = fileReference.Reference
		from.RawReference = from.Path
		src, err = oci.NewFromFS(ctx, os.DirFS(from.Path))
		if err != nil {
			return err
		}

		// Prepare destination
		fileReference.Registry = destination
		to := option.Target{
			Path:         path.Join(destination, fileReference.Repository),
			Reference:    fileReference.Reference,
			RawReference: fileReference.String(),
			Remote:       opts.To,
			Type:         option.TargetTypeRemote,
		}

		to.Reference = ""
		dst, err := to.NewTarget(opts.Common)
		if err != nil {
			return err
		}

		if err := to.EnsureReferenceNotEmpty(); err != nil {
			return err
		}

		_, err = oras.Copy(ctx, src, to.Reference, dst, to.Reference, uploadOptions)
		if err != nil {
			return err
		}

		fmt.Println("uploaded", to.AnnotatedReference(), "=>", to.AnnotatedReference())
	}

	return nil
}
