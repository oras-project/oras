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

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras/cmd/oras/internal/option"
)

type copyOptions struct {
	src option.Remote
	dst option.Remote
	option.Common
	rescursive bool

	srcRef string
	dstRef string
}

func copyCmd() *cobra.Command {
	var opts copyOptions
	opts.src.SetPrefix("source")
	opts.dst.SetPrefix("destination")
	opts.src.SetBlockPassStdin()
	opts.dst.SetBlockPassStdin()

	cmd := &cobra.Command{
		Use:     "copy <from-ref> <to-ref>",
		Aliases: []string{"cp"},
		Short:   "Copy manifests between repositories",
		Long: `Copy manifests between repositories

Examples - Copy the manifest tagged 'v1' from repository 'localhost:5000/net-monitor' to repository 'localhost:5000/net-monitor-copy' 
  oras cp localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1
Examples - Copy the manifest tagged 'v1' and referrer artifacts from repository 'localhost:5000/net-monitor' to 'localhost:5000/net-monitor-copy'
  oras cp -r localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1
`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.srcRef = args[0]
			opts.dstRef = args[1]
			return runCopy(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.rescursive, "recursive", "r", false, "recursively copy artifacts that reference the artifact being copied")

	return cmd
}

func runCopy(opts copyOptions) error {
	ctx, _ := opts.SetLoggerLevel()

	// Prepare source
	src, err := opts.src.NewRepository(opts.srcRef, opts.Common)
	if err != nil {
		return err
	}

	// Prepare destination
	dst, err := opts.dst.NewRepository(opts.dstRef, opts.Common)
	if err != nil {
		return err
	}

	// TODO: copy option

	// Copy
	srcRef := src.Reference
	dstRef := dst.Reference
	if dstRef.Reference == "" {
		dstRef.Reference = srcRef.ReferenceOrDefault()
	}
	var desc ocispec.Descriptor
	if opts.rescursive {
		desc, err = oras.ExtendedCopy(ctx, src, srcRef.ReferenceOrDefault(), dst, dstRef.ReferenceOrDefault(), oras.DefaultExtendedCopyOptions)
	} else {
		desc, err = oras.Copy(ctx, src, srcRef.ReferenceOrDefault(), dst, dstRef.ReferenceOrDefault(), oras.DefaultCopyOptions)
	}
	if err != nil {
		return err
	}

	fmt.Println("Copied", opts.srcRef, "=>", opts.dstRef)
	fmt.Println("Digest:", desc.Digest)

	return nil
}
