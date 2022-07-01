package main

import (
	"context"
	"fmt"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras/internal/trace"
)

type copyOptions struct {
	src        pullOptions
	dst        pushOptions
	rescursive bool
	verbose    bool

	debug bool
}

func copyCmd() *cobra.Command {
	var opts copyOptions
	cmd := &cobra.Command{
		Use:     "copy <from-ref> <to-ref>",
		Aliases: []string{"cp"},
		Short:   "Copy files from ref to ref",
		Long: `Copy artifacts from one reference to another reference
	# Examples 
	## Copy image only 
	oras cp localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1
	## Copy image and artifacts
	oras cp -r localhost:5000/net-monitor:v1 localhost:5000/net-monitor-copy:v1
		`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.src.targetRef = args[0]
			opts.dst.targetRef = args[1]
			return runCopy(opts)
		},
	}

	cmd.Flags().StringArrayVar(&opts.src.Remote.Configs, "from-config", nil, "source auth config path")
	cmd.Flags().StringVar(&opts.src.Remote.Username, "from-username", "", "source registry username")
	cmd.Flags().StringVar(&opts.src.Remote.Password, "from-password", "", "source registry password")
	cmd.Flags().BoolVar(&opts.src.Remote.Insecure, "from-insecure", false, "allow connections to SSL registry without certs")
	cmd.Flags().BoolVar(&opts.src.Remote.PlainHTTP, "from-plain-http", false, "use plain http and not https")

	cmd.Flags().StringArrayVar(&opts.dst.Remote.Configs, "to-config", nil, "target auth config path")
	cmd.Flags().StringVar(&opts.dst.Remote.Username, "to-username", "", "target registry username")
	cmd.Flags().StringVar(&opts.dst.Remote.Password, "to-password", "", "target registry password")
	cmd.Flags().BoolVar(&opts.dst.Remote.Insecure, "to-insecure", false, "allow connections to SSL registry without certs")
	cmd.Flags().BoolVar(&opts.dst.Remote.PlainHTTP, "to-plain-http", false, "use plain http and not https")

	cmd.Flags().BoolVarP(&opts.rescursive, "recursive", "r", false, "recursively copy artifacts that reference the artifact being copied")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "debug mode")

	return cmd
}

func runCopy(opts copyOptions) error {
	var logLevel logrus.Level
	if opts.debug {
		logLevel = logrus.DebugLevel
	} else if opts.verbose {
		logLevel = logrus.InfoLevel
	} else {
		logLevel = logrus.WarnLevel
	}
	ctx, _ := trace.WithLoggerLevel(context.Background(), logLevel)

	// Prepare source
	src, err := opts.src.NewRepository(opts.src.targetRef, opts.src.Common)
	if err != nil {
		return err
	}

	// Prepare destination
	dst, err := opts.dst.NewRepository(opts.dst.targetRef, opts.dst.Common)
	if err != nil {
		return err
	}

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

	fmt.Println("Copied", opts.src.targetRef, "=>", opts.dst.targetRef)
	fmt.Println("Digest:", desc.Digest)

	return nil
}
