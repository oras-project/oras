package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras/internal/credential"
	"oras.land/oras/internal/http"
	"oras.land/oras/internal/trace"
)

const (
	annotationConfig   = "$config"
	annotationManifest = "$manifest"
	tagStaged          = "staged"
)

type pushOptions struct {
	targetRef           string
	fileRefs            []string
	manifestConfigRef   string
	manifestAnnotations string
	artifactType        string
	artifactSubject     string
	verbose             bool

	debug     bool
	configs   []string
	username  string
	password  string
	insecure  bool
	plainHTTP bool
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

Example - Push file to the insecure registry:
  oras push localhost:5000/hello:latest hi.txt --insecure

Example - Push file to the HTTP registry:
  oras push localhost:5000/hello:latest hi.txt --plain-http
`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			opts.fileRefs = args[1:]
			return runPush(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.manifestConfigRef, "manifest-config", "", "", "manifest config file")
	cmd.Flags().StringVarP(&opts.manifestAnnotations, "manifest-annotations", "", "", "manifest annotation file")
	cmd.Flags().StringVarP(&opts.artifactType, "artifact-type", "", "", "artifact type")
	cmd.Flags().StringVarP(&opts.artifactSubject, "subject", "s", "", "subject artifact")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "debug mode")
	cmd.Flags().StringArrayVarP(&opts.configs, "config", "c", nil, "auth config path")
	cmd.Flags().StringVarP(&opts.username, "username", "u", "", "registry username")
	cmd.Flags().StringVarP(&opts.password, "password", "p", "", "registry password")
	cmd.Flags().BoolVarP(&opts.insecure, "insecure", "", false, "allow connections to SSL registry without certs")
	cmd.Flags().BoolVarP(&opts.plainHTTP, "plain-http", "", false, "use plain http and not https")
	return cmd
}

func runPush(opts pushOptions) error {
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
	remote, err := remote.NewRepository(opts.targetRef)
	if err != nil {
		return err
	}
	remote.PlainHTTP = opts.plainHTTP
	credStore, err := credential.NewStore(opts.configs...)
	if err != nil {
		return err
	}
	remote.Client = http.NewClient(http.ClientOptions{
		Credential:      credential.Credential(opts.username, opts.password),
		CredentialStore: credStore,
		SkipTLSVerify:   opts.insecure,
		Debug:           opts.debug,
	})

	// Load annotations
	var annotations map[string]map[string]string
	if opts.manifestAnnotations != "" {
		if err := decodeJSON(opts.manifestAnnotations, &annotations); err != nil {
			return err
		}
	}

	// Prepare manifest
	store := file.New("")
	defer store.Close()

	// Pack manifests
	if opts.artifactType != "" {
		err = packArtifact(ctx, remote, store, annotations, &opts)
	} else {
		err = packManifest(ctx, store, annotations, &opts)
	}
	if err != nil {
		return err
	}

	// ready to push
	target := &statusTracker{
		Target:  remote,
		out:     os.Stdout,
		prompt:  "Uploading",
		verbose: opts.verbose,
	}

	desc, err := oras.Copy(ctx, store, tagStaged, target, opts.targetRef)
	if err != nil {
		return err
	}

	fmt.Println("Pushed", opts.targetRef)
	fmt.Println("Digest:", desc.Digest)

	return nil
}

func packArtifact(ctx context.Context, remote content.Resolver, store *file.Store, annotations map[string]map[string]string, opts *pushOptions) error {
	subject, err := remote.Resolve(ctx, opts.artifactSubject)
	if err != nil {
		return err
	}
	files, err := loadFiles(ctx, store, annotations, opts)
	if err != nil {
		return err
	}

	manifest := artifactspec.Manifest{
		MediaType:    artifactspec.MediaTypeArtifactManifest,
		ArtifactType: opts.artifactType,
		Blobs:        ociToArtifactSlice(files),
		Subject:      ociToArtifact(subject),
		Annotations:  annotations[annotationManifest],
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	manifestDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    digest.FromBytes(manifestBytes),
		Size:      int64(len(manifestBytes)),
	}

	// store manifest
	if err := store.Push(ctx, manifestDesc, bytes.NewReader(manifestBytes)); err != nil && !errors.Is(err, errdef.ErrAlreadyExists) {
		return fmt.Errorf("failed to push manifest: %w", err)
	}
	return store.Tag(ctx, manifestDesc, tagStaged)
}

func packManifest(ctx context.Context, store *file.Store, annotations map[string]map[string]string, opts *pushOptions) error {
	var packOpts oras.PackOptions
	packOpts.ConfigAnnotations = annotations[annotationConfig]
	packOpts.ManifestAnnotations = annotations[annotationManifest]
	if opts.manifestConfigRef != "" {
		filename, mediaType := parseFileRef(opts.manifestConfigRef, ocispec.MediaTypeImageConfig)
		file, err := store.Add(ctx, annotationConfig, mediaType, filename)
		if err != nil {
			return err
		}
		file.Annotations = packOpts.ConfigAnnotations
		packOpts.ConfigDescriptor = &file
	}
	files, err := loadFiles(ctx, store, annotations, opts)
	if err != nil {
		return err
	}
	manifestDesc, err := oras.Pack(ctx, store, files, packOpts)
	if err != nil {
		return err
	}
	return store.Tag(ctx, manifestDesc, tagStaged)
}

func decodeJSON(filename string, v interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(v)
}

func loadFiles(ctx context.Context, store *file.Store, annotations map[string]map[string]string, opts *pushOptions) ([]ocispec.Descriptor, error) {
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
		file, err := store.Add(ctx, name, mediaType, filename)
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
	if len(files) == 0 {
		fmt.Println("Uploading empty artifact")
	}
	return files, nil
}

func ociToArtifactSlice(descs []ocispec.Descriptor) []artifactspec.Descriptor {
	res := make([]artifactspec.Descriptor, 0, len(descs))
	for _, desc := range descs {
		res = append(res, ociToArtifact(desc))
	}
	return res
}

func ociToArtifact(desc ocispec.Descriptor) artifactspec.Descriptor {
	return artifactspec.Descriptor{
		MediaType:   desc.MediaType,
		Digest:      desc.Digest,
		Size:        desc.Size,
		URLs:        desc.URLs,
		Annotations: desc.Annotations,
	}
}
