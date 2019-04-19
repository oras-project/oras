package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/deislabs/oras/pkg/content"
	ctxo "github.com/deislabs/oras/pkg/context"
	"github.com/deislabs/oras/pkg/oras"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/reference"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type pullOptions struct {
	targetRef          string
	allowedMediaTypes  []string
	allowAllMediaTypes bool
	keepOldFiles       bool
	pathTraversal      bool
	output             string
	verbose            bool

	debug    bool
	configs  []string
	username string
	password string
}

func pullCmd() *cobra.Command {
	var opts pullOptions
	cmd := &cobra.Command{
		Use:   "pull <name:tag|name@digest>",
		Short: "Pull files from remote registry",
		Long: `Pull files from remote registry

Example - Pull only files with the "application/vnd.oci.image.layer.v1.tar" media type (default):
  oras pull localhost:5000/hello:latest

Example - Pull only files with the custom "application/vnd.me.hi" media type:
  oras pull localhost:5000/hello:latest -t application/vnd.me.hi

Example - Pull all files, any media type:
  oras pull localhost:5000/hello:latest -a
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return runPull(opts)
		},
	}

	cmd.Flags().StringArrayVarP(&opts.allowedMediaTypes, "media-type", "t", nil, "allowed media types to be pulled")
	cmd.Flags().BoolVarP(&opts.allowAllMediaTypes, "allow-all", "a", false, "allow all media types to be pulled")
	cmd.Flags().BoolVarP(&opts.keepOldFiles, "keep-old-files", "k", false, "do not replace existing files when pulling, treat them as errors")
	cmd.Flags().BoolVarP(&opts.pathTraversal, "allow-path-traversal", "T", false, "allow storing files out of the output directory")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "output directory")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "verbose output")

	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "debug mode")
	cmd.Flags().StringArrayVarP(&opts.configs, "config", "c", nil, "auth config path")
	cmd.Flags().StringVarP(&opts.username, "username", "u", "", "registry username")
	cmd.Flags().StringVarP(&opts.password, "password", "p", "", "registry password")
	return cmd
}

func runPull(opts pullOptions) error {
	ctx := context.Background()
	if opts.debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else if !opts.verbose {
		ctx = ctxo.WithLoggerDiscarded(ctx)
	}
	if opts.allowAllMediaTypes {
		opts.allowedMediaTypes = nil
	} else if len(opts.allowedMediaTypes) == 0 {
		opts.allowedMediaTypes = []string{content.DefaultBlobMediaType, content.DefaultBlobDirMediaType}
	}

	resolver := newResolver(opts.username, opts.password, opts.configs...)
	store := content.NewFileStore(opts.output)
	defer store.Close()
	store.DisableOverwrite = opts.keepOldFiles
	store.AllowPathTraversalOnWrite = opts.pathTraversal

	desc, _, err := oras.Pull(ctx, resolver, opts.targetRef, store,
		oras.WithAllowedMediaTypes(opts.allowedMediaTypes),
		oras.WithPullCallbackHandler(pullStatusTrack()),
	)
	if err != nil {
		if err == reference.ErrObjectRequired {
			return fmt.Errorf("image reference format is invalid. Please specify <name:tag|name@digest>")
		}
		return err
	}
	fmt.Println("Pulled", opts.targetRef)
	fmt.Println("Digest:", desc.Digest)

	return nil
}

func pullStatusTrack() images.Handler {
	var printLock sync.Mutex
	return images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		if name, ok := content.ResolveName(desc); ok {
			digestString := desc.Digest.String()
			if err := desc.Digest.Validate(); err == nil {
				if algo := desc.Digest.Algorithm(); algo == digest.SHA256 {
					digestString = desc.Digest.Encoded()[:12]
				}
			}
			printLock.Lock()
			defer printLock.Unlock()
			fmt.Println("Downloaded", digestString, name)
		}
		return nil, nil
	})
}
