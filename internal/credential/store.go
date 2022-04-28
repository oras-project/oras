package credential

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// Store provides credential CRUD operations.
type Store struct {
	configs []*configfile.ConfigFile
}

// NewStore generates a store based on the passed in config file path.
func NewStore(configPaths ...string) (*Store, error) {
	if len(configPaths) == 0 {
		// No config path passed, load default docker config file.
		cfg, err := config.Load(config.Dir())
		if err != nil {
			return nil, err
		}
		if !cfg.ContainsAuth() {
			cfg.CredentialsStore = credentials.DetectDefaultStore(cfg.CredentialsStore)
		}

		return &Store{
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

	return &Store{
		configs: configs,
	}, nil
}

// loadConfigFile reads the configuration files from the given path.
func loadConfigFile(path string) (*configfile.ConfigFile, error) {
	cfg := configfile.New(path)
	if _, err := os.Stat(path); err == nil {
		// if the config file already exists, load from file
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

// Store retrieves a credential for a given registry.
func (s *Store) Store(registry string, cred auth.Credential) error {
	authConf := types.AuthConfig{
		Username:      cred.Username,
		Password:      cred.Password,
		ServerAddress: registry,
		IdentityToken: cred.RefreshToken,
		RegistryToken: cred.AccessToken,
	}
	for _, c := range s.configs {
		return c.GetCredentialsStore(registry).Store(authConf)
	}
	return nil
}

// Store erase a credential for a given registry.
func (s *Store) Erase(registry string) error {
	for _, c := range s.configs {
		return c.GetCredentialsStore(registry).Erase(registry)
	}
	return nil
}

// Credential specifies the function for resolving the credential for the
// given registry (i.e. host:port).
// `EmptyCredential` is a valid return value and should not be considered as
// an error.
// If nil, the credential is always resolved to `EmptyCredential`.
func (s *Store) Credential(ctx context.Context, registry string) (auth.Credential, error) {
	for _, c := range s.configs {
		authConf, _ := c.GetCredentialsStore(registry).Get(registry)
		return auth.Credential{
			Username:     authConf.Username,
			Password:     authConf.Password,
			AccessToken:  authConf.RegistryToken,
			RefreshToken: authConf.IdentityToken,
		}, nil

	}
	return auth.EmptyCredential, nil
}
