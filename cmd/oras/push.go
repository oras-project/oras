package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/deislabs/oras/pkg/content"
	ctxo "github.com/deislabs/oras/pkg/context"
	"github.com/deislabs/oras/pkg/oras"

	"github.com/containerd/containerd/images"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	annotationConfig   = "$config"
	annotationManifest = "$manifest"
)

type pushOptions struct {
	targetRef              string
	fileRefs               []string
	manifestConfigRef      string
	manifestAnnotations    string
	pathValidationDisabled bool
	verbose                bool

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

Example - Push file "hi.txt" with the custom "application/vnd.me.hi" media type:
  oras push localhost:5000/hello:latest hi.txt:application/vnd.me.hi

Example - Push multiple files with different media types:
  oras push localhost:5000/hello:latest hi.txt:application/vnd.me.hi bye.txt:application/vnd.me.bye

Example - Push file "hi.txt" with the custom manifest config "config.json" of the custom "application/vnd.me.config" media type:
  oras push --manifest-config config.json:application/vnd.me.config localhost:5000/hello:latest hi.txt
`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			opts.fileRefs = args[1:]
			return runPush(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.manifestConfigRef, "manifest-config", "", "", "manifest config file")
	cmd.Flags().StringVarP(&opts.manifestAnnotations, "manifest-annotations", "", "", "manifest annotation file")
	cmd.Flags().BoolVarP(&opts.pathValidationDisabled, "disable-path-validation", "", false, "skip path validation")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "debug mode")
	cmd.Flags().StringArrayVarP(&opts.configs, "config", "c", nil, "auth config path")
	cmd.Flags().StringVarP(&opts.username, "username", "u", "", "registry username")
	cmd.Flags().StringVarP(&opts.password, "password", "p", "", "registry password")
	return cmd
}

func runPush(opts pushOptions) error {
	ctx := context.Background()
	if opts.debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else if !opts.verbose {
		ctx = ctxo.WithLoggerDiscarded(ctx)
	}

	// load files
	var (
		annotations map[string]map[string]string
		store       = content.NewFileStore("")
		pushOpts    []oras.PushOpt
	)
	defer store.Close()
	if opts.manifestAnnotations != "" {
		if err := decodeJSON(opts.manifestAnnotations, &annotations); err != nil {
			return err
		}
		if value, ok := annotations[annotationConfig]; ok {
			pushOpts = append(pushOpts, oras.WithConfigAnnotations(value))
		}
		if value, ok := annotations[annotationManifest]; ok {
			pushOpts = append(pushOpts, oras.WithManifestAnnotations(value))
		}
	}
	if opts.manifestConfigRef != "" {
		filename, mediaType := parseFileRef(opts.manifestConfigRef, ocispec.MediaTypeImageConfig)
		file, err := store.Add(annotationConfig, mediaType, filename)
		if err != nil {
			return err
		}
		file.Annotations = nil
		pushOpts = append(pushOpts, oras.WithConfig(file))
	}
	if opts.pathValidationDisabled {
		pushOpts = append(pushOpts, oras.WithNameValidation(nil))
	}
	files, err := loadFiles(store, annotations, &opts)
	if err != nil {
		return err
	}

	// ready to push
	resolver := newResolver(opts.username, opts.password, opts.configs...)
	pushOpts = append(pushOpts, oras.WithPushBaseHandler(pushStatusTrack()))
	desc, err := oras.Push(ctx, resolver, opts.targetRef, store, files, pushOpts...)
	if err != nil {
		return err
	}

	fmt.Println("Pushed", opts.targetRef)
	fmt.Println("Digest:", desc.Digest)

	return nil
}

func decodeJSON(filename string, v interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(v)
}

func loadFiles(store *content.FileStore, annotations map[string]map[string]string, opts *pushOptions) ([]ocispec.Descriptor, error) {
	var files []ocispec.Descriptor
	for _, fileRef := range opts.fileRefs {
		filename, mediaType := parseFileRef(fileRef, "")
		name := filepath.Clean(filename)
		if !filepath.IsAbs(name) {
			// convert to slash-separated path unless it is absolute path
			name = filepath.ToSlash(name)
		}
		if opts.verbose {
			fmt.Println("Preparing", name)
		}
		file, err := store.Add(name, mediaType, filename)
		if err != nil {
			return nil, err
		}
		if annotations != nil {
			if value, ok := annotations[filename]; ok {
				if file.Annotations == nil {
					file.Annotations = value
				} else {
					for k, v := range value {
						file.Annotations[k] = v
					}
				}
			}
		}
		files = append(files, file)
	}
	return files, nil
}

func pushStatusTrack() images.Handler {
	var printLock sync.Mutex
	return images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		if name, ok := content.ResolveName(desc); ok {
			printLock.Lock()
			defer printLock.Unlock()
			fmt.Println("Uploading", desc.Digest.Encoded()[:12], name)
		}
		return nil, nil
	})
}
