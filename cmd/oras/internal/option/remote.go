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
	ctls "crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	nhttp "net/http"
	"os"
	"strings"

	"github.com/spf13/pflag"
	oremote "oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras/internal/credential"
	"oras.land/oras/internal/http"
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
}

// ApplyFlags applies flags to a command flag set.
func (remote *Remote) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringArrayVarP(&remote.Configs, "config", "c", nil, "auth config path")
	fs.StringVarP(&remote.Username, "username", "u", "", "registry username")
	fs.StringVarP(&remote.Password, "password", "p", "", "registry password or identity token")
	fs.BoolVarP(&remote.PasswordFromStdin, "password-stdin", "", false, "read password or identity token from stdin")
	fs.BoolVarP(&remote.Insecure, "insecure", "k", false, "allow connections to SSL registry without certs")
	fs.StringVarP(&remote.CACertFilePath, "ca-file", "", "", "server certificate authority file for the remote registry")
	fs.BoolVarP(&remote.PlainHTTP, "plain-http", "", false, "allow insecure connections to registry without SSL")
}

// ReadPassword tries to read password with optional cmd prompt.
func (remote *Remote) ReadPassword() (err error) {
	if remote.Password != "" {
		fmt.Fprintln(os.Stderr, "WARNING! Using --password via the CLI is insecure. Use --password-stdin.")
	} else if remote.PasswordFromStdin {
		// Prompt for credential
		password, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		remote.Password = strings.TrimSuffix(string(password), "\n")
		remote.Password = strings.TrimSuffix(remote.Password, "\r")
	}
	return nil
}

// tlsConfig assembles the tls config.
func (remote *Remote) tlsConfig() (config *ctls.Config, err error) {
	config = &ctls.Config{}
	var caPool *x509.CertPool
	if remote.CACertFilePath == "" {
		caPool = nil
	} else if caPool, err = http.LoadCertPool(remote.CACertFilePath); err != nil {
		return nil, err
	}

	config.RootCAs = caPool
	config.InsecureSkipVerify = remote.Insecure
	return
}

// AuthClient assembles a oras auth client
func (remote *Remote) AuthClient(debug bool) (client *auth.Client, err error) {
	config, err := remote.tlsConfig()
	if err != nil {
		return nil, err
	}
	client = &auth.Client{
		Client: &nhttp.Client{
			Transport: &nhttp.Transport{
				TLSClientConfig: config,
			},
		},
	}
	client.SetUserAgent("oras/" + version.GetVersion())
	if debug {
		client.Client.Transport = trace.NewTransport(client.Client.Transport)
	}

	cred := credential.Credential(remote.Username, remote.Password)
	if cred != auth.EmptyCredential {
		client.Credential = func(ctx context.Context, s string) (auth.Credential, error) {
			return remote.Credential(), nil
		}
	} else {
		store, err := credential.NewStore(remote.Configs...)
		if err != nil {
			return nil, err
		}
		client.Credential = store.Credential
	}
	return
}

// Credential returns a credential based on the remote options.
func (remote *Remote) Credential() auth.Credential {
	return credential.Credential(remote.Username, remote.Password)
}

// NewRegistry assembles a oras remote registry.
func (remote *Remote) NewRegistry(hostname string, common Common) (reg *oremote.Registry, err error) {
	reg, err = oremote.NewRegistry(hostname)
	if err != nil {
		return nil, err
	}
	reg.PlainHTTP = remote.PlainHTTP
	if reg.Client, err = remote.AuthClient(common.Debug); err != nil {
		return nil, err
	}
	return
}
