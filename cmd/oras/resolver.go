package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"

	auth "github.com/deislabs/oras/pkg/auth/docker"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
)

func newResolver(username, password string, insecure bool, plainHTTP bool, configs ...string) (remotes.Resolver, *docker.ResolverOptions) {
	header := http.Header{}
	header.Set("Accept", strings.Join([]string{
		images.MediaTypeDockerSchema2Manifest,
		images.MediaTypeDockerSchema2ManifestList,
		ocispec.MediaTypeImageManifest,
		ocispec.MediaTypeImageIndex,
		artifactspec.MediaTypeArtifactManifest,
		"*/*",
	}, ", "))
	opts := &docker.ResolverOptions{
		PlainHTTP: plainHTTP,
		Headers:   header,
	}

	client := http.DefaultClient
	if insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}
	opts.Client = client

	if username != "" || password != "" {
		opts.Credentials = func(hostName string) (string, string, error) {
			return username, password, nil
		}
		return docker.NewResolver(*opts), opts
	}
	cli, err := auth.NewClient(configs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Error loading auth file: %v\n", err)
	}
	dockerCli := cli.(*auth.Client)
	opts.Credentials = dockerCli.Credential
	return docker.NewResolver(*opts), opts
}
