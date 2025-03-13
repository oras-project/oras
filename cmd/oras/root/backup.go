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
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras/cmd/oras/internal/command"
	"oras.land/oras/cmd/oras/internal/display"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/docker"
	"oras.land/oras/internal/graph"
	"oras.land/oras/internal/registryutil"
	"path/filepath"
	"slices"
)

type backupOptions struct {
	option.Cache
	option.Common
	option.Platform

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
	ctx, _ := command.GetLogger(cmd, &opts.Common)
	for _, source := range opts.sources {
		target, err := opts.From.GetNewTarget(cmd, source)
		if err != nil {
			return err
		}
		//repo.PlainHTTP = opts.isPlainHttp(registry)
		//repo.HandleWarning = opts.handleWarning(registry, logger)
		//if repo.Client, err = opts.authClient(registry, common.Debug); err != nil {
		//	return nil, err
		//}
		//repo.SkipReferrersGC = true
		//if opts.ReferrersAPI != nil {
		//	if err := repo.SetReferrersCapability(*opts.ReferrersAPI); err != nil {
		//		return nil, err
		//	}
		//}

		src, err := opts.CachedTarget(target)
		if err != nil {
			return err
		}

		rOpts := oras.DefaultResolveOptions
		rOpts.TargetPlatform = opts.Platform.Platform
		desc, err := oras.Resolve(ctx, src, opts.From.Reference, rOpts)
		if err != nil {
			return fmt.Errorf("failed to resolve %s: %w", opts.From.Reference, err)
		}

		destination := filepath.Join(opts.output, target.Reference.Repository)
		dst, err := oci.New(destination)
		if err != nil {
			return err
		}
		ctx = registryutil.WithScopeHint(ctx, dst, auth.ActionPull, auth.ActionPush)

		err = doBackup(ctx, desc, target, dst, opts)
		if err != nil {
			return err
		}

		if from, err := digest.Parse(opts.From.Reference); err == nil && from != desc.Digest {
			// correct source digest
			opts.From.RawReference = fmt.Sprintf("%s@%s", opts.From.Path, desc.Digest.String())
		}
		_ = opts.Printer.Println("Copied", opts.From.AnnotatedReference(), "=>[%s] %s", option.TargetTypeOCILayout, opts.output)
		_ = opts.Printer.Println("Digest:", desc.Digest)
	}

	return nil
}

func doBackup(ctx context.Context, desc ocispec.Descriptor, src oras.ReadOnlyGraphTarget, dst oras.GraphTarget, opts *backupOptions) (err error) {

	backupHandler, _ := display.NewCopyHandler(opts.Printer, opts.TTY, dst)

	// Prepare backup options
	extendedCopyOptions := oras.DefaultExtendedCopyOptions
	extendedCopyOptions.Concurrency = opts.concurrency
	extendedCopyOptions.FindPredecessors = func(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		return registry.Referrers(ctx, src, desc, "")
	}

	dst, err = backupHandler.StartTracking(dst)
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

	err = recursiveBackup(ctx, src, dst, opts.output, desc, extendedCopyOptions)
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
