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
	opts.src.SetMark("from-")
	opts.dst.SetMark("to-")

	opts.src.SetBlockPassStdin()
	opts.dst.SetBlockPassStdin()

	cmd := &cobra.Command{
		Use:     "copy <from-ref> <to-ref>",
		Aliases: []string{"cp"},
		Short:   "Copy files from ref to ref",
		Long: `Copy artifacts from one reference to another reference

Examples - Copy image only 
  oras cp localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1
Examples - Copy image and artifacts
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
		desc, err = oras.ExtendedCopy(ctx,
			src,
			srcRef.ReferenceOrDefault(),
			dst,
			dstRef.ReferenceOrDefault(),
			oras.DefaultExtendedCopyOptions,
		)
	} else {
		desc, err = oras.Copy(ctx,
			src,
			srcRef.ReferenceOrDefault(),
			dst,
			dstRef.ReferenceOrDefault(),
			oras.DefaultCopyOptions,
		)
	}
	if err != nil {
		return err
	}

	fmt.Println("Copied", opts.srcRef, "=>", opts.dstRef)
	fmt.Println("Digest:", desc.Digest)

	return nil
}
