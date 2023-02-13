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
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras/internal/credential"
	"oras.land/oras/internal/crypto"
	onet "oras.land/oras/internal/net"
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

	resolveFlag           []string
	resolveDialContext    func(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error)
	applyDistributionSpec bool
	distributionSpec      distributionSpec
	headerFlags           []string
	headers               http.Header
}

// EnableDistributionSpecFlag set distribution specification flag as applicable.
func (opts *Remote) EnableDistributionSpecFlag() {
	opts.applyDistributionSpec = true
}

// ApplyFlags applies flags to a command flag set.
func (opts *Remote) ApplyFlags(fs *pflag.FlagSet) {
	opts.ApplyFlagsWithPrefix(fs, "", "")
	fs.BoolVarP(&opts.PasswordFromStdin, "password-stdin", "", false, "read password or identity token from stdin")
	fs.StringArrayVarP(&opts.headerFlags, "header", "H", []string{}, "add custom headers to requests")
}

func applyPrefix(prefix, description string) (flagPrefix, notePrefix string) {
	if prefix == "" {
		return "", ""
	}
	return prefix + "-", description + " "
}

// ApplyFlagsWithPrefix applies flags to a command flag set with a prefix string.
// Commonly used for non-unary remote targets.
func (opts *Remote) ApplyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	var (
		shortUser     string
		shortPassword string
		flagPrefix    string
		notePrefix    string
	)
	if prefix == "" {
		shortUser, shortPassword = "u", "p"
	}
	flagPrefix, notePrefix = applyPrefix(prefix, description)

	if opts.applyDistributionSpec {
		opts.distributionSpec.ApplyFlagsWithPrefix(fs, prefix, description)
	}
	fs.StringVarP(&opts.Username, flagPrefix+"username", shortUser, "", notePrefix+"registry username")
	fs.StringVarP(&opts.Password, flagPrefix+"password", shortPassword, "", notePrefix+"registry password or identity token")
	fs.BoolVarP(&opts.Insecure, flagPrefix+"insecure", "", false, "allow connections to "+notePrefix+"SSL registry without certs")
	fs.BoolVarP(&opts.PlainHTTP, flagPrefix+"plain-http", "", false, "allow insecure connections to "+notePrefix+"registry without SSL check")
	fs.StringVarP(&opts.CACertFilePath, flagPrefix+"ca-file", "", "", "server certificate authority file for the remote "+notePrefix+"registry")

	if fs.Lookup("registry-config") == nil {
		fs.StringArrayVarP(&opts.Configs, "registry-config", "", nil, "`path` of the authentication file")
	}

	if fs.Lookup("resolve") == nil {
		fs.StringArrayVarP(&opts.resolveFlag, "resolve", "", nil, "customized DNS formatted in `host:port:address`")
	}
}

// Parse tries to read password with optional cmd prompt.
func (opts *Remote) Parse() error {
	if err := opts.parseCustomHeaders(); err != nil {
		return err
	}
	if err := opts.readPassword(); err != nil {
		return err
	}
	return opts.distributionSpec.Parse()
}

// readPassword tries to read password with optional cmd prompt.
func (opts *Remote) readPassword() (err error) {
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

// parseResolve parses resolve flag.
func (opts *Remote) parseResolve() error {
	if len(opts.resolveFlag) == 0 {
		return nil
	}

	formatError := func(param, message string) error {
		return fmt.Errorf("failed to parse resolve flag %q: %s", param, message)
	}
	var dialer onet.Dialer
	for _, r := range opts.resolveFlag {
		parts := strings.SplitN(r, ":", 3)
		if len(parts) < 3 {
			return formatError(r, "expecting host:port:address")
		}

		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return formatError(r, "expecting uint64 port")
		}

		// ipv6 zone is not parsed
		to := net.ParseIP(parts[2])
		if to == nil {
			return formatError(r, "invalid IP address")
		}
		dialer.Add(parts[0], port, to)
	}
	opts.resolveDialContext = func(base *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
		dialer.Dialer = base
		return dialer.DialContext
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
func (opts *Remote) authClient(registry string, debug bool) (client *auth.Client, err error) {
	config, err := opts.tlsConfig()
	if err != nil {
		return nil, err
	}
	if err := opts.parseResolve(); err != nil {
		return nil, err
	}
	resolveDialContext := opts.resolveDialContext
	if resolveDialContext == nil {
		resolveDialContext = func(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
			return dialer.DialContext
		}
	}
	client = &auth.Client{
		Client: &http.Client{
			// default value are derived from http.DefaultTransport
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: resolveDialContext(&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}),
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				TLSClientConfig:       config,
			},
		},
		Cache:  auth.NewCache(),
		Header: opts.headers,
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
		// For a user case with a registry from 'docker.io', the hostname is "registry-1.docker.io"
		// According to the the behavior of Docker CLI,
		// credential under key "https://index.docker.io/v1/" should be provided
		if registry == "docker.io" {
			client.Credential = func(ctx context.Context, hostname string) (auth.Credential, error) {
				if hostname == "registry-1.docker.io" {
					hostname = "https://index.docker.io/v1/"
				}
				return store.Credential(ctx, hostname)
			}
		} else {
			client.Credential = store.Credential
		}
	}
	return
}

func (opts *Remote) parseCustomHeaders() error {
	if len(opts.headerFlags) != 0 {
		headers := map[string][]string{}
		for _, h := range opts.headerFlags {
			name, value, found := strings.Cut(h, ":")
			if !found || strings.TrimSpace(name) == "" {
				return fmt.Errorf("invalid header: %q", h)
			}
			headers[name] = append(headers[name], value)
		}
		opts.headers = headers
	}
	return nil
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
	hostname = reg.Reference.Registry
	reg.PlainHTTP = opts.isPlainHttp(hostname)
	if reg.Client, err = opts.authClient(hostname, common.Debug); err != nil {
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
	hostname := repo.Reference.Registry
	repo.PlainHTTP = opts.isPlainHttp(hostname)
	if repo.Client, err = opts.authClient(hostname, common.Debug); err != nil {
		return nil, err
	}
	if opts.distributionSpec.referrersAPI != nil {
		repo.SetReferrersCapability(*opts.distributionSpec.referrersAPI)
	}
	return
}

// isPlainHttp returns the plain http flag for a given registry.
func (opts *Remote) isPlainHttp(registry string) bool {
	host, _, _ := net.SplitHostPort(registry)
	if host == "localhost" || registry == "localhost" {
		return true
	}
	return opts.PlainHTTP
}
