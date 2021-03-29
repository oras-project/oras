package main

import (
	"context"
	"errors"
	"fmt"

	ctxo "github.com/deislabs/oras/pkg/context"
	"github.com/deislabs/oras/pkg/oras"

	"github.com/containerd/containerd/reference"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type discoverOptions struct {
	targetRef    string
	artifactType string
	verbose      bool

	debug     bool
	configs   []string
	username  string
	password  string
	insecure  bool
	plainHTTP bool
}

func discoverCmd() *cobra.Command {
	var opts discoverOptions
	cmd := &cobra.Command{
		Use:   "discover [options] <name:tag|name@digest>",
		Short: "discover artifacts from remote registry",
		Long: `discover artifacts from remote registry

Example - Discover artifacts of type "" linked with the specified reference:
  oras discover --artifact-type application/vnd.cncf.notary.v2 localhost:5000/hello:latest
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.artifactType == "" {
				return errors.New("artifact type not specified")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return runDiscover(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.artifactType, "artifact-type", "", "", "artifact type")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "verbose output")

	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "debug mode")
	cmd.Flags().StringArrayVarP(&opts.configs, "config", "c", nil, "auth config path")
	cmd.Flags().StringVarP(&opts.username, "username", "u", "", "registry username")
	cmd.Flags().StringVarP(&opts.password, "password", "p", "", "registry password")
	cmd.Flags().BoolVarP(&opts.insecure, "insecure", "", false, "allow connections to SSL registry without certs")
	cmd.Flags().BoolVarP(&opts.plainHTTP, "plain-http", "", false, "use plain http and not https")
	return cmd
}

func runDiscover(opts discoverOptions) error {
	ctx := context.Background()
	if opts.debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else if !opts.verbose {
		ctx = ctxo.WithLoggerDiscarded(ctx)
	}

	resolver := newResolver(opts.username, opts.password, opts.insecure, opts.plainHTTP, opts.configs...)

	desc, artifacts, err := oras.Discover(ctx, resolver, opts.targetRef, opts.artifactType)
	if err != nil {
		if err == reference.ErrObjectRequired {
			return fmt.Errorf("image reference format is invalid. Please specify <name:tag|name@digest>")
		}
		return err
	}

	fmt.Println("Discovered", len(artifacts), "artifacts referencing", opts.targetRef)
	fmt.Println("Digest:", desc.Digest)
	for _, artifact := range artifacts {
		fmt.Println("Reference:", artifact.ArtifactType)
		for _, blob := range artifact.Blobs {
			fmt.Println("-", blob.Digest)
		}
	}

	return nil
}
