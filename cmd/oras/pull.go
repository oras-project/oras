package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras/internal/credential"
	"oras.land/oras/internal/http"
	"oras.land/oras/internal/trace"
)

type pullOptions struct {
	targetRef     string
	keepOldFiles  bool
	pathTraversal bool
	output        string
	verbose       bool

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
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return runPull(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.keepOldFiles, "keep-old-files", "k", false, "do not replace existing files when pulling, treat them as errors")
	cmd.Flags().BoolVarP(&opts.pathTraversal, "allow-path-traversal", "T", false, "allow storing files out of the output directory")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "output directory")
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
	var logLevel logrus.Level
	if opts.debug {
		logLevel = logrus.DebugLevel
	} else if opts.verbose {
		logLevel = logrus.InfoLevel
	} else {
		logLevel = logrus.WarnLevel
	}
	ctx, _ := trace.WithLoggerLevel(context.Background(), logLevel)

	// Prepare client
	repo, err := remote.NewRepository(opts.targetRef)
	if err != nil {
		return err
	}
	setPlainHTTP(repo, opts.plainHTTP)
	credStore, err := credential.NewStore(opts.configs...)
	if err != nil {
		return err
	}
	repo.Client = http.NewClient(http.ClientOptions{
		Credential:      credential.Credential(opts.username, opts.password),
		CredentialStore: credStore,
		SkipTLSVerify:   opts.insecure,
		Debug:           opts.debug,
	})

	// Prepare target
	store := file.New(opts.output)
	defer store.Close()
	store.DisableOverwrite = opts.keepOldFiles
	store.AllowPathTraversalOnWrite = opts.pathTraversal

	tracker := &statusTracker{
		Target:     store,
		out:        os.Stdout,
		printAfter: true,
		prompt:     "Downloaded",
		verbose:    opts.verbose,
	}

	tag := repo.Reference.ReferenceOrDefault()
	desc, err := oras.Copy(ctx, repo, tag, tracker, "")
	if err != nil {
		return err
	}
	fmt.Println("Pulled", opts.targetRef)
	fmt.Println("Digest:", desc.Digest)

	return nil
}
