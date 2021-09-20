/*
   Copyright The containerd Authors.

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

package docker

import (
	"context"
	"io"
	"net/http"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
	"github.com/pkg/errors"
)

func (d *dockerDiscoverer) Fetcher(ctx context.Context, ref string) (remotes.Fetcher, error) {
	d.reference = ref
	return d, nil
}

func (d *dockerDiscoverer) Fetch(ctx context.Context, desc ocispec.Descriptor) (io.ReadCloser, error) {
	hosts, err := d.filterHosts(docker.HostCapabilityPull)
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return nil, errors.Wrap(errdefs.ErrNotFound, "no pull hosts")
	}

	switch desc.MediaType {
	case artifactspec.MediaTypeArtifactManifest:
		var errs []error
		for _, host := range hosts {
			req := d.request(host, http.MethodGet, "manifests", desc.Digest.String())
			if err := req.addNamespace(d.refspec.Hostname()); err != nil {
				errs = append(errs, err)
				continue
			}

			resp, err := req.doWithRetries(ctx, nil)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			if resp.StatusCode > 299 {
				resp.Body.Close()
				continue
			}

			return resp.Body, nil
		}

		if len(errs) > 1 {
			for _, e := range errs {
				log.G(ctx).WithError(e).Errorf("error fetching artifact manifest")
			}
			return nil, errs[0]
		}

		log.G(ctx).WithField("media-type", desc.MediaType).WithField("digest", desc.Digest).Warnf("Could not fetch artifacts manifest")

		// Since we couldn't get the manifest, fallback to the original resolver behavior to allow any error handling to happen
	}

	fetcher, err := d.Resolver.Fetcher(ctx, d.reference)
	if err != nil {
		return nil, err
	}

	return fetcher.Fetch(ctx, desc)
}
