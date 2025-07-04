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
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras/cmd/oras/internal/command"
	"oras.land/oras/cmd/oras/internal/display"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/docker"
	"oras.land/oras/internal/graph"
	"oras.land/oras/internal/listener"
	"oras.land/oras/internal/registryutil"
	"slices"
)

type backupOptions struct {
	option.Cache
	option.Common
	option.Platform
	option.Terminal

	From        option.Target
	output      string
	concurrency int
	sources     []string
}

func backupCmd() *cobra.Command {
	var opts backupOptions
	cmd := &cobra.Command{
		Use:   "backup [flags] --output <directory> <source>{:<tag>|@<digest>}...",
		Short: "Backup artifacts to a file",
		Long: `Backup artifacts from a source to disk. When backing up an image index, all of its manifests will be copied

Example - Backup artifacts from a registry to disk:
  oras backup --output ./mirror registry.k8s.io/kube-apiserver-arm64:v1.31.0 registry.k8s.io/kube-controller-manager-arm64:v1.31.0
`,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.sources = args
			opts.From.RawReference = args[0]
			opts.DisableTTY(opts.Debug, false)
			return option.Parse(cmd, &opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackup(cmd, &opts)
		},
	}
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "output directory")
	_ = cmd.MarkFlagRequired("output")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 3, "concurrency level")
	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.From)
}

func runBackup(cmd *cobra.Command, opts *backupOptions) error {
	ctx, logger := command.GetLogger(cmd, &opts.Common)

	dst := option.NewOCITarget(opts.output)
	err := dst.Parse(cmd)
	if err != nil {
		return fmt.Errorf("parse target: %w", err)
	}
	ctx = registryutil.WithScopeHint(ctx, dst, auth.ActionPull, auth.ActionPush)

	to, err := dst.NewTarget(opts.Common, logger)
	if err != nil {
		return err
	}

	for _, source := range opts.sources {
		src, err := opts.From.GetRemoteRepository(cmd, source)
		if err != nil {
			return err
		}

		err = doBackup(ctx, src, dst, to, opts)
		if err != nil {
			return err
		}
	}

	return nil
}

func doBackup(ctx context.Context, src *remote.Repository, dst *option.Target, to oras.GraphTarget, opts *backupOptions) (err error) {
	rOpts := oras.DefaultResolveOptions
	rOpts.TargetPlatform = opts.Platform.Platform
	desc, err := oras.Resolve(ctx, src, opts.From.Reference, rOpts)
	if err != nil {
		return fmt.Errorf("failed to resolve %s: %w", opts.From.Reference, err)
	}

	// Prepare backup options
	extendedCopyOptions := oras.DefaultExtendedCopyOptions
	extendedCopyOptions.Concurrency = opts.concurrency
	extendedCopyOptions.FindPredecessors = func(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		return registry.Referrers(ctx, src, desc, "")
	}

	backupHandler, metadataHandler := display.NewBackupHandler(opts.Printer, opts.TTY, to)
	to, err = backupHandler.StartTracking(to)
	if err != nil {
		return err
	}
	defer func() {
		stopErr := backupHandler.StopTracking()
		if err == nil {
			err = stopErr
		}
	}()
	extendedCopyOptions.OnCopySkipped = backupHandler.OnCopySkipped
	extendedCopyOptions.PreCopy = backupHandler.PreCopy
	extendedCopyOptions.PostCopy = backupHandler.PostCopy
	extendedCopyOptions.OnMounted = backupHandler.OnMounted

	err = recursiveBackup(ctx, src, to, opts.output, desc, extendedCopyOptions)
	if err != nil {
		return err
	}

	if from, err := digest.Parse(opts.From.Reference); err == nil && from != desc.Digest {
		// correct source digest
		opts.From.RawReference = fmt.Sprintf("%s@%s", opts.From.Path, desc.Digest.String())
	}

	err = metadataHandler.OnCopied(opts.From.GetDisplayReference(), dst.GetDisplayReference())
	if err != nil {
		return err
	}

	tagListener := listener.NewTaggedListener(to, metadataHandler.OnTagged)
	//tag := src.Reference.Repository
	//if opts.From.Reference != desc.Digest.String() {
	//	tag += ":" + opts.From.Reference
	//}
	_, err = oras.Tag(ctx, tagListener, desc.Digest.String(), opts.From.RawReference)
	return err
}

// recursiveBackup copies an artifact and its referrers from one target to another.
// If the artifact is a manifest list or index, referrers of its manifests are copied as well.
func recursiveBackup(ctx context.Context, src oras.ReadOnlyGraphTarget, dst oras.Target, dstRef string, root ocispec.Descriptor, opts oras.ExtendedCopyOptions) error {
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

	var err error
	if dstRef == "" || dstRef == root.Digest.String() {
		err = oras.ExtendedCopyGraph(ctx, src, dst, root, opts.ExtendedCopyGraphOptions)
	} else {
		_, err = oras.ExtendedCopy(ctx, src, root.Digest.String(), dst, dstRef, opts)
	}
	return err
}
