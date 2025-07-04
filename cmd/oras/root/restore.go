/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package root

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras/cmd/oras/internal/command"
	"oras.land/oras/cmd/oras/internal/display"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/docker"
	"oras.land/oras/internal/graph"
	"oras.land/oras/internal/listener"
	"oras.land/oras/internal/registryutil"
	"path/filepath"
	"slices"
)

type restoreOptions struct {
	option.Cache
	option.Common
	option.Platform
	option.Terminal

	To          option.Target
	Registry    string
	input       string
	concurrency int
}

func restoreCmd() *cobra.Command {
	var opts restoreOptions
	cmd := &cobra.Command{
		Use:   "restore [flags] --input <directory> <registry>",
		Short: "Restore artifacts to a file",
		Long: `Restore artifacts disk to a registry. When restoring an image index, all of its manifests will be copied

Example - Restore artifacts from a registry to disk:
  oras restore --input ./mirror localhost:15000
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.DisableTTY(opts.Debug, false)
			opts.Registry = args[0]
			opts.To.RawReference = filepath.Join(opts.Registry, "restore:v1")
			return option.Parse(cmd, &opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestore(cmd, &opts)
		},
	}
	cmd.Flags().StringVarP(&opts.input, "input", "", "", "input directory")
	_ = cmd.MarkFlagRequired("input")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 3, "concurrency level")
	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.To)
}

func runRestore(cmd *cobra.Command, opts *restoreOptions) error {
	ctx, logger := command.GetLogger(cmd, &opts.Common)
	from := option.NewOCITarget(opts.input)
	err := from.Parse(cmd)
	if err != nil {
		return fmt.Errorf("parse source target %s: %w", opts.input, err)
	}

	src, err := from.NewTarget(opts.Common, logger)
	if err != nil {
		return fmt.Errorf("failed to create target %s: %v", from.Path, err)
	}

	tagReader, err := from.NewReadonlyTarget(ctx, opts.Common, logger)
	if err != nil {
		return fmt.Errorf("failed to create target %s: %v", from.Path, err)
	}

	err = tagReader.Tags(ctx, "", func(tags []string) error {
		for _, tag := range tags {
			from.Reference = tag
			ctx = registryutil.WithScopeHint(ctx, from, auth.ActionPull, auth.ActionPush)

			reference, err := registry.ParseReference(tag)
			if err != nil {
				logger.Warnf("Skipping tag with parse error %s: %v\n", tag, err)
				continue
			}
			if reference.Reference == "" {
				continue
			}

			opts.To.Path = ""
			opts.To.RawReference = filepath.Join(opts.Registry, reference.Repository+":"+reference.Reference)
			err = opts.To.Parse(cmd)
			if err != nil {
				logger.Warn("Skipping destination with parse error: %v\n", err)
				continue
			}

			err = doRestore(ctx, from, src, &opts.To, opts, logger)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func doRestore(ctx context.Context, from *option.Target, src oras.GraphTarget, to *option.Target, opts *restoreOptions, logger logrus.FieldLogger) (err error) {
	dst, err := to.NewTarget(opts.Common, logger)
	if err != nil {
		return fmt.Errorf("failed to create target %s: %v", to.Path, err)
	}

	rOpts := oras.DefaultResolveOptions
	rOpts.TargetPlatform = opts.Platform.Platform
	desc, err := oras.Resolve(ctx, src, from.Reference, rOpts)
	if err != nil {
		return fmt.Errorf("failed to resolve %v %s: %v", src, from.Path, err)
	}

	// what is this silly thing
	_, err = opts.CachedTarget(dst)
	if err != nil {
		return err
	}

	// Prepare restore options
	extendedCopyOptions := oras.DefaultExtendedCopyOptions
	extendedCopyOptions.Concurrency = opts.concurrency
	extendedCopyOptions.FindPredecessors = func(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		return registry.Referrers(ctx, src, desc, "")
	}

	restoreHandler, metadataHandler := display.NewRestoreHandler(opts.Printer, opts.TTY, dst)
	dst, err = restoreHandler.StartTracking(dst)
	if err != nil {
		return err
	}
	defer func() {
		stopErr := restoreHandler.StopTracking()
		if err == nil {
			err = stopErr
		}
	}()
	extendedCopyOptions.OnCopySkipped = restoreHandler.OnCopySkipped
	extendedCopyOptions.PreCopy = restoreHandler.PreCopy
	extendedCopyOptions.PostCopy = restoreHandler.PostCopy
	extendedCopyOptions.OnMounted = restoreHandler.OnMounted

	err = recursiveRestore(ctx, src, dst, desc.Digest.String(), desc, extendedCopyOptions)
	if err != nil {
		return err
	}

	if from, err := digest.Parse(to.Path); err == nil && from != desc.Digest {
		// correct source digest
		to.Path = fmt.Sprintf("%s@%s", to.Path, desc.Digest.String())
	}

	err = metadataHandler.OnCopied(from.GetDisplayReference(), to.GetDisplayReference())
	if err != nil {
		return err
	}

	tagListener := listener.NewTaggedListener(dst, metadataHandler.OnTagged)
	_, err = oras.Tag(ctx, tagListener, desc.Digest.String(), to.Reference)
	return err
}

// recursiveRestore copies an artifact and its referrers from one target to another.
// If the artifact is a manifest list or index, referrers of its manifests are copied as well.
func recursiveRestore(ctx context.Context, src oras.ReadOnlyGraphTarget, dst oras.Target, dstRef string, root ocispec.Descriptor, opts oras.ExtendedCopyOptions) (err error) {
	if root.MediaType == ocispec.MediaTypeImageIndex || root.MediaType == docker.MediaTypeManifestList {
		fetched, err := content.FetchAll(ctx, src, root)
		if err != nil {
			return err
		}
		var index ocispec.Index
		if err = json.Unmarshal(fetched, &index); err != nil {
			return nil
		}

		referrers, err := graph.FindPredecessors(ctx, src, index.Manifests, opts)
		if err != nil {
			return err
		}
		referrers = slices.DeleteFunc(referrers, func(desc ocispec.Descriptor) bool {
			return content.Equal(desc, root)
		})

		findPredecessor := opts.FindPredecessors
		opts.FindPredecessors = func(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
			descs, err := findPredecessor(ctx, src, desc)
			if err != nil {
				return nil, err
			}
			if content.Equal(desc, root) {
				// make sure referrers of child manifests are copied by pointing them to root
				descs = append(descs, referrers...)
			}
			return descs, nil
		}
	}

	return oras.ExtendedCopyGraph(ctx, src, dst, root, opts.ExtendedCopyGraphOptions)
}
