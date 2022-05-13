package main

import (
	"context"
	"fmt"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras/internal/credential"
	"oras.land/oras/internal/http"
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

	cmd.Flags().StringArrayVar(&opts.src.configs, "from-config", nil, "source auth config path")
	cmd.Flags().StringVar(&opts.src.username, "from-username", "", "source registry username")
	cmd.Flags().StringVar(&opts.src.password, "from-password", "", "source registry password")
	cmd.Flags().BoolVar(&opts.src.insecure, "from-insecure", false, "allow connections to SSL registry without certs")
	cmd.Flags().BoolVar(&opts.src.plainHTTP, "from-plain-http", false, "use plain http and not https")

	cmd.Flags().StringArrayVar(&opts.dst.configs, "to-config", nil, "target auth config path")
	cmd.Flags().StringVar(&opts.dst.username, "to-username", "", "target registry username")
	cmd.Flags().StringVar(&opts.dst.password, "to-password", "", "target registry password")
	cmd.Flags().BoolVar(&opts.dst.insecure, "to-insecure", false, "allow connections to SSL registry without certs")
	cmd.Flags().BoolVar(&opts.dst.plainHTTP, "to-plain-http", false, "use plain http and not https")

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
	src, err := remote.NewRepository(opts.src.targetRef)
	if err != nil {
		return err
	}
	setPlainHTTP(src, opts.src.plainHTTP)
	credStore, err := credential.NewStore(opts.src.configs...)
	if err != nil {
		return err
	}
	src.Client = http.NewClient(http.ClientOptions{
		Credential:      credential.Credential(opts.src.username, opts.src.password),
		CredentialStore: credStore,
		SkipTLSVerify:   opts.src.insecure,
		Debug:           opts.debug,
	})

	// Prepare destination
	dst, err := remote.NewRepository(opts.dst.targetRef)
	if err != nil {
		return err
	}
	setPlainHTTP(dst, opts.dst.plainHTTP)
	credStore, err = credential.NewStore(opts.dst.configs...)
	if err != nil {
		return err
	}
	dst.Client = http.NewClient(http.ClientOptions{
		Credential:      credential.Credential(opts.dst.username, opts.dst.password),
		CredentialStore: credStore,
		SkipTLSVerify:   opts.dst.insecure,
		Debug:           opts.debug,
	})

	// Copy
	tracker := &statusTracker{
		Target:     dst,
		out:        os.Stdout,
		printAfter: true,
		prompt:     "Copied",
		verbose:    opts.verbose,
	}

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
			tracker,
			dstRef.ReferenceOrDefault(),
		)
	} else {
		desc, err = oras.Copy(ctx,
			src,
			srcRef.ReferenceOrDefault(),
			tracker,
			dstRef.ReferenceOrDefault(),
		)
	}
	if err != nil {
		return err
	}

	fmt.Println("Copied", opts.src.targetRef, "=>", opts.dst.targetRef)
	fmt.Println("Digest:", desc.Digest)

	return nil
}
