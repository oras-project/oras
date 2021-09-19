package docker

import (
	"net/http"
	"strings"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/oras/internal/version"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// NewResolver returns a new resolver to a Docker registry
func NewOpts(options *docker.ResolverOptions) *docker.ResolverOptions {
	if options.Tracker == nil {
		options.Tracker = docker.NewInMemoryTracker()
	}

	if options.Headers == nil {
		options.Headers = make(http.Header)
	}
	if _, ok := options.Headers["User-Agent"]; !ok {
		options.Headers.Set("User-Agent", "oras/"+version.Version)
	}

	resolveHeader := http.Header{}
	if _, ok := options.Headers["Accept"]; !ok {
		// set headers for all the types we support for resolution.
		resolveHeader.Set("Accept", strings.Join([]string{
			images.MediaTypeDockerSchema2Manifest,
			images.MediaTypeDockerSchema2ManifestList,
			ocispec.MediaTypeImageManifest,
			ocispec.MediaTypeImageIndex, "*/*"}, ", "))
	} else {
		resolveHeader["Accept"] = options.Headers["Accept"]
		delete(options.Headers, "Accept")
	}

	if options.Hosts == nil {
		opts := []docker.RegistryOpt{}
		if options.Host != nil {
			opts = append(opts, docker.WithHostTranslator(options.Host))
		}

		if options.Authorizer == nil {
			options.Authorizer = docker.NewDockerAuthorizer(
				docker.WithAuthClient(options.Client),
				docker.WithAuthHeader(options.Headers),
				docker.WithAuthCreds(options.Credentials))
		}
		opts = append(opts, docker.WithAuthorizer(options.Authorizer))

		if options.Client != nil {
			opts = append(opts, docker.WithClient(options.Client))
		}
		if options.PlainHTTP {
			opts = append(opts, docker.WithPlainHTTP(docker.MatchAllHosts))
		} else {
			opts = append(opts, docker.WithPlainHTTP(docker.MatchLocalhost))
		}
		options.Hosts = docker.ConfigureDefaultRegistries(opts...)
	}

	return options
}
