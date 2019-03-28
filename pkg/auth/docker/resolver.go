package docker

import (
	"context"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/registry"
)

// Resolver returns a new authenticated resolver.
func (c *Client) Resolver(_ context.Context) (remotes.Resolver, error) {
	credential := func(hostName string) (string, string, error) {
		if hostName == registry.DefaultV2Registry.Host {
			hostName = registry.IndexServer
		}
		var (
			auth types.AuthConfig
			err  error
		)
		for _, cfg := range c.configs {
			auth, err = cfg.GetAuthConfig(hostName)
			if err != nil {
				// fall back to next config
				continue
			}
			if auth.IdentityToken != "" {
				return "", auth.IdentityToken, nil
			}
			if auth.Username == "" && auth.Password == "" {
				// fall back to next config
				continue
			}
			return auth.Username, auth.Password, nil
		}
		return "", "", err
	}
	return docker.NewResolver(docker.ResolverOptions{
		Credentials: credential,
	}), nil
}
