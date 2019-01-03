package main

import (
	"context"
	"strings"

	"github.com/shizhMSFT/oras/pkg/content"
	"github.com/shizhMSFT/oras/pkg/oras"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type pushOptions struct {
	targetRef string
	fileRefs  []string

	debug    bool
	username string
	password string
}

func pushCmd() *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:   "push name[:tag|@digest] file [file...]",
		Short: "Push files to remote registry",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			opts.fileRefs = args[1:]
			return runPush(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "debug mode")
	cmd.Flags().StringVarP(&opts.username, "username", "u", "", "registry username")
	cmd.Flags().StringVarP(&opts.password, "password", "p", "", "registry password")
	return cmd
}

func runPush(opts pushOptions) error {
	if opts.debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	resolver := newResolver(opts.username, opts.password)

	var (
		files []ocispec.Descriptor
		store = content.NewFileStore("")
	)
	for _, fileRef := range opts.fileRefs {
		ref := strings.SplitN(fileRef, ":", 2)
		filename := ref[0]
		var mediaType string
		if len(ref) == 2 {
			mediaType = ref[1]
		}
		file, err := store.Add(filename, mediaType, "")
		if err != nil {
			return err
		}
		files = append(files, file)
	}

	return oras.Push(context.Background(), resolver, opts.targetRef, store, files)
}
