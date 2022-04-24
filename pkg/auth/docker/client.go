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

package docker

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
	"github.com/moby/moby/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras/pkg/auth"
	iface "oras.land/oras/pkg/auth"
)

// Client provides authentication operations for docker registries.
type Client struct {
	configs []*configfile.ConfigFile
}

// NewClient creates a new auth client based on provided config paths.
// If not config path is provided, the default path is used.
// Credentials are read from the first config and fall backs to next.
// All changes will only be written to the first config file.
func NewClient(configPaths ...string) (auth.Client, error) {
	if len(configPaths) == 0 {
		cfg, err := config.Load(config.Dir())
		if err != nil {
			return nil, err
		}
		if !cfg.ContainsAuth() {
			cfg.CredentialsStore = credentials.DetectDefaultStore(cfg.CredentialsStore)
		}

		return &Client{
			configs: []*configfile.ConfigFile{cfg},
		}, nil
	}

	var configs []*configfile.ConfigFile
	for _, path := range configPaths {
		cfg, err := loadConfigFile(path)
		if err != nil {
			return nil, fmt.Errorf(path, err)
		}
		configs = append(configs, cfg)
	}

	return &Client{
		configs: configs,
	}, nil
}

func (c *Client) primaryCredentialsStore(hostname string) credentials.Store {
	return c.configs[0].GetCredentialsStore(hostname)
}

// loadConfigFile reads the configuration files from the given path.
func loadConfigFile(path string) (*configfile.ConfigFile, error) {
	cfg := configfile.New(path)
	if _, err := os.Stat(path); err == nil {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		if err := cfg.LoadFromReader(file); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if !cfg.ContainsAuth() {
		cfg.CredentialsStore = credentials.DetectDefaultStore(cfg.CredentialsStore)
	}
	return cfg, nil
}

// Logout logs out from a docker registry identified by the hostname.
func (c *Client) Logout(_ context.Context, hostname string) error {
	hostname = resolveHostname(hostname)

	var configs []*configfile.ConfigFile
	for _, config := range c.configs {
		if _, ok := config.AuthConfigs[hostname]; ok {
			configs = append(configs, config)
		}
	}
	if len(configs) == 0 {
		return auth.ErrNotLoggedIn
	}

	// Log out form the primary config only as backups are read-only.
	return c.primaryCredentialsStore(hostname).Erase(hostname)
}

// Login logs in to a docker registry identified by the hostname with custom
// options.
func (c *Client) Login(settings *iface.LoginSettings) error {
	hostname := resolveHostname(settings.Hostname)
	cred := types.AuthConfig{
		Username:      settings.Username,
		ServerAddress: hostname,
	}
	if settings.Username == "" {
		cred.IdentityToken = settings.Secret
	} else {
		cred.Password = settings.Secret
	}

	// Login to ensure valid credential
	remote, err := remote.NewRegistry(settings.Hostname)
	if err != nil {
		return err
	}
	remote.PlainHTTP = settings.PlainHTTP
	remote.Client = settings.GetAuthClient()
	if err = remote.Ping(settings.Context); err != nil {
		return err
	}

	// Store credential
	return c.primaryCredentialsStore(hostname).Store(cred)

}

// resolveHostname resolves Docker specific hostnames
func resolveHostname(hostname string) string {
	switch hostname {
	case registry.IndexHostname, registry.IndexName, registry.DefaultV2Registry.Host:
		return registry.IndexServer
	}
	return hostname
}

// LoadCredential loads the username and secret for a certain remote server.
// Returns empty strings if the remote server is not logged in.
func (c *Client) LoadCredential(ctx context.Context, hostname string) (username, secret string) {
	for _, config := range c.configs {
		authConfig, err := config.GetAuthConfig(hostname)
		if err == nil {
			if authConfig.IdentityToken != "" {
				return "", authConfig.IdentityToken
			}
			return authConfig.Username, authConfig.Password
		}
	}
	return
}
