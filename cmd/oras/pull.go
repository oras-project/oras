package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/reference"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras-go/pkg/artifact"
	"oras.land/oras-go/pkg/content"
	ctxo "oras.land/oras-go/pkg/context"
	"oras.land/oras-go/pkg/oras"
)

type pullOptions struct {
	targetRef         string
	keepOldFiles      bool
	pathTraversal     bool
	output            string
	manifestConfigRef string
	verbose           bool
	cacheRoot         string

	debug     bool
	configs   []string
	username  string
	password  string
	insecure  bool
	plainHTTP bool
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

Example - Pull files from the insecure registry:
  oras pull localhost:5000/hello:latest --insecure

Example - Pull files from the HTTP registry:
  oras pull localhost:5000/hello:latest --plain-http

Example - Pull files with local cache:
  export ORAS_CACHE=~/.oras/cache
  oras pull localhost:5000/hello:latest
`,
		Args: cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			opts.cacheRoot = os.Getenv("ORAS_CACHE")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return runPull(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.keepOldFiles, "keep-old-files", "k", false, "do not replace existing files when pulling, treat them as errors")
	cmd.Flags().BoolVarP(&opts.pathTraversal, "allow-path-traversal", "T", false, "allow storing files out of the output directory")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "output directory")
	cmd.Flags().StringVarP(&opts.manifestConfigRef, "manifest-config", "", "", "output manifest config file")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "verbose output")

	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "debug mode")
	cmd.Flags().StringArrayVarP(&opts.configs, "config", "c", nil, "auth config path")
	cmd.Flags().StringVarP(&opts.username, "username", "u", "", "registry username")
	cmd.Flags().StringVarP(&opts.password, "password", "p", "", "registry password")
	cmd.Flags().BoolVarP(&opts.insecure, "insecure", "", false, "allow connections to SSL registry without certs")
	cmd.Flags().BoolVarP(&opts.plainHTTP, "plain-http", "", false, "use plain http and not https")
	return cmd
}

func runPull(opts pullOptions) error {
	ctx := context.Background()
	if opts.debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else if !opts.verbose {
		ctx = ctxo.WithLoggerDiscarded(ctx)
	}
	resolver := newResolver(opts.username, opts.password, opts.insecure, opts.plainHTTP, opts.configs...)
	store := content.NewFileStore(opts.output)
	defer store.Close()
	store.DisableOverwrite = opts.keepOldFiles
	store.AllowPathTraversalOnWrite = opts.pathTraversal

	pullOpts := []oras.PullOpt{
		oras.WithAllowedMediaTypes([]string{}),
		oras.WithPullStatusTrack(os.Stdout),
	}
	if opts.cacheRoot != "" {
		cachedStore, err := newStoreWithCache(store, opts.cacheRoot)
		if err != nil {
			return err
		}
		pullOpts = append(pullOpts, oras.WithContentProvideIngester(cachedStore))
	}
	if opts.manifestConfigRef != "" {
		pullOpts = appendPullManifestConfigHandlers(pullOpts, opts.manifestConfigRef)
	}

	desc, artifacts, err := oras.Pull(ctx, resolver, opts.targetRef, store, pullOpts...)
	if err != nil {
		if err == reference.ErrObjectRequired {
			return fmt.Errorf("image reference format is invalid. Please specify <name:tag|name@digest>")
		}
		return err
	}
	if len(artifacts) == 0 {
		fmt.Println("Downloaded empty artifact")
	}
	fmt.Println("Pulled", opts.targetRef)
	fmt.Println("Digest:", desc.Digest)

	return nil
}

func appendPullManifestConfigHandlers(pullOpts []oras.PullOpt, manifestConfigRef string) []oras.PullOpt {
	filename, mediaType := parseFileRef(manifestConfigRef, artifact.UnknownConfigMediaType)

	var pullOnce sync.Once
	marker := images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) (children []ocispec.Descriptor, err error) {
		if desc.MediaType == mediaType {
			pullOnce.Do(func() {
				if desc.Annotations != nil {
					delete(desc.Annotations, ocispec.AnnotationTitle)
				}
				annotations := make(map[string]string)
				for k, v := range desc.Annotations {
					annotations[k] = v
				}
				annotations[ocispec.AnnotationTitle] = filename
				desc.Annotations = annotations
				children = []ocispec.Descriptor{desc}
			})
		}
		return
	})
	stopper := images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		if desc.MediaType == mediaType {
			if name, _ := content.ResolveName(desc); name == "" {
				return nil, images.ErrStopHandler
			}
		}
		return nil, nil
	})

	return append(pullOpts,
		oras.WithPullEmptyNameAllowed(),
		oras.WithAllowedMediaType(mediaType),
		oras.WithPullBaseHandler(marker, stopper),
	)
}
