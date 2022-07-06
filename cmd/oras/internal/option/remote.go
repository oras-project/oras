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

package option

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras/internal/credential"
	"oras.land/oras/internal/crypto"
	"oras.land/oras/internal/trace"
	"oras.land/oras/internal/version"
)

// Remote options struct.
type Remote struct {
	CACertFilePath    string
	PlainHTTP         bool
	Insecure          bool
	Configs           []string
	Username          string
	PasswordFromStdin bool
	Password          string

	notePrefix     string
	flagPrefix     string
	blockPassStdin bool
}

// ApplyFlags applies flags to a command flag set.
func (opts *Remote) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&opts.Username, opts.flagPrefix+"username", "u", "", opts.notePrefix+"registry username")
	fs.StringVarP(&opts.Password, opts.flagPrefix+"password", "p", "", opts.notePrefix+"registry password or identity token")
	fs.BoolVarP(&opts.Insecure, opts.flagPrefix+"insecure", "", false, "allow connections to "+opts.notePrefix+"SSL registry without certs")
	fs.BoolVarP(&opts.PlainHTTP, opts.flagPrefix+"plain-http", "", false, "allow insecure connections to "+opts.notePrefix+"registry without SSL")
	fs.StringVarP(&opts.CACertFilePath, opts.flagPrefix+"ca-file", "", "", "server certificate authority file for the remote "+opts.notePrefix+"registry")

	fs.StringArrayVarP(&opts.Configs, "config", "c", nil, "auth config path")
	if !opts.blockPassStdin {
		fs.BoolVarP(&opts.PasswordFromStdin, "password-stdin", "", false, "read password or identity token from stdin")
	}
}

// SetPrefix sets prefix for applicable flags.
func (opts *Remote) SetPrefix(target string) {
	opts.flagPrefix = target + "-"
	opts.notePrefix = target + " "
}

// SetBlockPassStdin disables Password input from Stdin.
func (opts *Remote) SetBlockPassStdin() {
	opts.blockPassStdin = true
}

// ReadPassword tries to read password with optional cmd prompt.
func (opts *Remote) ReadPassword() (err error) {
	if opts.Password != "" {
		fmt.Fprintln(os.Stderr, "WARNING! Using --password via the CLI is insecure. Use --password-stdin.")
	} else if opts.PasswordFromStdin {
		// Prompt for credential
		password, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		opts.Password = strings.TrimSuffix(string(password), "\n")
		opts.Password = strings.TrimSuffix(opts.Password, "\r")
	}
	return nil
}

// tlsConfig assembles the tls config.
func (opts *Remote) tlsConfig() (*tls.Config, error) {
	config := &tls.Config{
		InsecureSkipVerify: opts.Insecure,
	}
	if opts.CACertFilePath != "" {
		var err error
		config.RootCAs, err = crypto.LoadCertPool(opts.CACertFilePath)
		if err != nil {
			return nil, err
		}
	}
	return config, nil
}

// authClient assembles a oras auth client.
func (opts *Remote) authClient(debug bool) (client *auth.Client, err error) {
	config, err := opts.tlsConfig()
	if err != nil {
		return nil, err
	}
	client = &auth.Client{
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: config,
			},
		},
		Cache: auth.NewCache(),
	}
	client.SetUserAgent("oras/" + version.GetVersion())
	if debug {
		client.Client.Transport = trace.NewTransport(client.Client.Transport)
	}

	cred := opts.Credential()
	if cred != auth.EmptyCredential {
		client.Credential = func(ctx context.Context, s string) (auth.Credential, error) {
			return cred, nil
		}
	} else {
		store, err := credential.NewStore(opts.Configs...)
		if err != nil {
			return nil, err
		}
		client.Credential = store.Credential
	}
	return
}

// Credential returns a credential based on the remote options.
func (opts *Remote) Credential() auth.Credential {
	return credential.Credential(opts.Username, opts.Password)
}

// NewRegistry assembles a oras remote registry.
func (opts *Remote) NewRegistry(hostname string, common Common) (reg *remote.Registry, err error) {
	reg, err = remote.NewRegistry(hostname)
	if err != nil {
		return nil, err
	}
	reg.PlainHTTP = opts.isPlainHttp(reg.Reference.Registry)
	if reg.Client, err = opts.authClient(common.Debug); err != nil {
		return nil, err
	}
	return
}

// NewRepository assembles a oras remote repository.
func (opts *Remote) NewRepository(reference string, common Common) (repo *remote.Repository, err error) {
	repo, err = remote.NewRepository(reference)
	if err != nil {
		return nil, err
	}
	repo.PlainHTTP = opts.isPlainHttp(repo.Reference.Registry)
	if repo.Client, err = opts.authClient(common.Debug); err != nil {
		return nil, err
	}
	return
}

// isPlainHttp returns the plain http flag for a given regsitry.
func (opts *Remote) isPlainHttp(registry string) bool {
	host, _, _ := net.SplitHostPort(registry)
	if host == "localhost" || registry == "localhost" {
		return true
	}
	return opts.PlainHTTP
}
