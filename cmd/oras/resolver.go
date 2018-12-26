package main

import (
	"os"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/docker/cli/cli/config"
	"github.com/docker/docker/registry"
)

func newResolver(username, password string) remotes.Resolver {
	cfg := config.LoadDefaultConfigFile(os.Stderr)
	credential := func(hostName string) (string, string, error) {
		if hostName == registry.DefaultV2Registry.Host {
			hostName = registry.IndexServer
		}
		auth, err := cfg.GetAuthConfig(hostName)
		if err != nil {
			return "", "", err
		}
		if auth.IdentityToken != "" {
			return "", auth.IdentityToken, nil
		}
		return auth.Username, auth.Password, nil
	}
	if username != "" || password != "" {
		credential = func(hostName string) (string, string, error) {
			return username, password, nil
		}
	}
	return docker.NewResolver(docker.ResolverOptions{
		Credentials: credential,
	})
}
