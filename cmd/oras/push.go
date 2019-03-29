package main

import (
	"context"
	"strings"

	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type pushOptions struct {
	targetRef string
	fileRefs  []string

	debug    bool
	configs  []string
	username string
	password string
}

func pushCmd() *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:   "push name[:tag|@digest] file[:type] [file...]",
		Short: "Push files to remote registry",
		Long: `Push files to remote registry

Example - Push file "hi.txt" with the "application/vnd.oci.image.layer.v1.tar" media type (default):
  oras push localhost:5000/hello:latest hi.txt

Example - Pull file "hi.txt" with the custom "application/vnd.me.hi" media type:
  oras push localhost:5000/hello:latest hi.txt:application/vnd.me.hi

Example - Push multiple files with different media types:
  oras push localhost:5000/hello:latest hi.txt:application/vnd.me.hi bye.txt:application/vnd.me.bye
`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			opts.fileRefs = args[1:]
			return runPush(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "debug mode")
	cmd.Flags().StringArrayVarP(&opts.configs, "config", "c", nil, "auth config path")
	cmd.Flags().StringVarP(&opts.username, "username", "u", "", "registry username")
	cmd.Flags().StringVarP(&opts.password, "password", "p", "", "registry password")
	return cmd
}

func runPush(opts pushOptions) error {
	if opts.debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	resolver := newResolver(opts.username, opts.password, opts.configs...)

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
