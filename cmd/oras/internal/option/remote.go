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
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/errcode"
	"oras.land/oras-go/v2/registry/remote/retry"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/internal/credential"
	"oras.land/oras/internal/crypto"
	onet "oras.land/oras/internal/net"
	"oras.land/oras/internal/trace"
	"oras.land/oras/internal/version"
)

// Remote options struct contains flags and arguments specifying one registry.
// Remote implements oerrors.Handler and interface.
type Remote struct {
	DistributionSpec
	CACertFilePath    string
	Insecure          bool
	Configs           []string
	Username          string
	PasswordFromStdin bool
	Password          string

	resolveFlag           []string
	applyDistributionSpec bool
	headerFlags           []string
	headers               http.Header
	warned                map[string]*sync.Map
	plainHTTP             func() (plainHTTP bool, enforced bool)
	store                 credentials.Store
}

// EnableDistributionSpecFlag set distribution specification flag as applicable.
func (opts *Remote) EnableDistributionSpecFlag() {
	opts.applyDistributionSpec = true
}

// ApplyFlags applies flags to a command flag set.
func (opts *Remote) ApplyFlags(fs *pflag.FlagSet) {
	opts.ApplyFlagsWithPrefix(fs, "", "")
	fs.BoolVarP(&opts.PasswordFromStdin, "password-stdin", "", false, "read password or identity token from stdin")
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
		shortHeader   string
		flagPrefix    string
		notePrefix    string
	)
	if prefix == "" {
		shortUser, shortPassword = "u", "p"
		shortHeader = "H"
	}
	flagPrefix, notePrefix = applyPrefix(prefix, description)

	if opts.applyDistributionSpec {
		opts.DistributionSpec.ApplyFlagsWithPrefix(fs, prefix, description)
	}
	fs.StringVarP(&opts.Username, flagPrefix+"username", shortUser, "", notePrefix+"registry username")
	fs.StringVarP(&opts.Password, flagPrefix+"password", shortPassword, "", notePrefix+"registry password or identity token")
	fs.BoolVarP(&opts.Insecure, flagPrefix+"insecure", "", false, "allow connections to "+notePrefix+"SSL registry without certs")
	plainHTTPFlagName := flagPrefix + "plain-http"
	plainHTTP := fs.Bool(plainHTTPFlagName, false, "allow insecure connections to "+notePrefix+"registry without SSL check")
	opts.plainHTTP = func() (bool, bool) {
		return *plainHTTP, fs.Changed(plainHTTPFlagName)
	}
	fs.StringVarP(&opts.CACertFilePath, flagPrefix+"ca-file", "", "", "server certificate authority file for the remote "+notePrefix+"registry")
	fs.StringArrayVarP(&opts.resolveFlag, flagPrefix+"resolve", "", nil, "customized DNS for "+notePrefix+"registry, formatted in `host:port:address[:address_port]`")
	fs.StringArrayVarP(&opts.Configs, flagPrefix+"registry-config", "", nil, "`path` of the authentication file for "+notePrefix+"registry")
	fs.StringArrayVarP(&opts.headerFlags, flagPrefix+"header", shortHeader, nil, "add custom headers to "+notePrefix+"requests")
}

// Parse tries to read password with optional cmd prompt.
func (opts *Remote) Parse() error {
	if err := opts.parseCustomHeaders(); err != nil {
		return err
	}
	return opts.readPassword()
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
func (opts *Remote) parseResolve(baseDial onet.DialFunc) (onet.DialFunc, error) {
	if len(opts.resolveFlag) == 0 {
		return baseDial, nil
	}

	formatError := func(param, message string) error {
		return fmt.Errorf("failed to parse resolve flag %q: %s", param, message)
	}
	var dialer onet.Dialer
	for _, r := range opts.resolveFlag {
		parts := strings.SplitN(r, ":", 4)
		length := len(parts)
		if length < 3 {
			return nil, formatError(r, "expecting host:port:address[:address_port]")
		}
		host := parts[0]
		hostPort, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, formatError(r, "expecting uint64 host port")
		}
		// ipv6 zone is not parsed
		address := net.ParseIP(parts[2])
		if address == nil {
			return nil, formatError(r, "invalid IP address")
		}
		addressPort := hostPort
		if length > 3 {
			addressPort, err = strconv.Atoi(parts[3])
			if err != nil {
				return nil, formatError(r, "expecting uint64 address port")
			}
		}
		dialer.Add(host, hostPort, address, addressPort)
	}
	dialer.BaseDialContext = baseDial
	return dialer.DialContext, nil
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
	baseTransport := http.DefaultTransport.(*http.Transport).Clone()
	baseTransport.TLSClientConfig = config
	dialContext, err := opts.parseResolve(baseTransport.DialContext)
	if err != nil {
		return nil, err
	}
	baseTransport.DialContext = dialContext
	client = &auth.Client{
		Client: &http.Client{
			// http.RoundTripper with a retry using the DefaultPolicy
			// see: https://pkg.go.dev/oras.land/oras-go/v2/registry/remote/retry#Policy
			Transport: retry.NewTransport(baseTransport),
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
		var err error
		opts.store, err = credential.NewStore(opts.Configs...)
		if err != nil {
			return nil, err
		}
		client.Credential = credentials.Credential(opts.store)
	}
	return
}

// ConfigPath returns the config path of the credential store.
func (opts *Remote) ConfigPath() (string, error) {
	if opts.store == nil {
		return "", errors.New("no credential store initialized")
	}
	if ds, ok := opts.store.(*credentials.DynamicStore); ok {
		return ds.ConfigPath(), nil
	}
	return "", errors.New("store doesn't support getting config path")
}

func (opts *Remote) parseCustomHeaders() error {
	if len(opts.headerFlags) != 0 {
		headers := map[string][]string{}
		for _, h := range opts.headerFlags {
			name, value, found := strings.Cut(h, ":")
			if !found || strings.TrimSpace(name) == "" {
				// In conformance to the RFC 2616 specification
				// Reference: https://www.rfc-editor.org/rfc/rfc2616#section-4.2
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

func (opts *Remote) handleWarning(registry string, logger logrus.FieldLogger) func(warning remote.Warning) {
	if opts.warned == nil {
		opts.warned = make(map[string]*sync.Map)
	}
	warned := opts.warned[registry]
	if warned == nil {
		warned = &sync.Map{}
		opts.warned[registry] = warned
	}
	logger = logger.WithField("registry", registry)
	return func(warning remote.Warning) {
		if _, loaded := warned.LoadOrStore(warning.WarningValue, struct{}{}); !loaded {
			logger.Warn(warning.Text)
		}
	}
}

// NewRegistry assembles a oras remote registry.
func (opts *Remote) NewRegistry(registry string, common Common, logger logrus.FieldLogger) (reg *remote.Registry, err error) {
	reg, err = remote.NewRegistry(registry)
	if err != nil {
		return nil, err
	}
	registry = reg.Reference.Registry
	reg.PlainHTTP = opts.isPlainHttp(registry)
	reg.HandleWarning = opts.handleWarning(registry, logger)
	if reg.Client, err = opts.authClient(registry, common.Debug); err != nil {
		return nil, err
	}
	return
}

// NewRepository assembles a oras remote repository.
func (opts *Remote) NewRepository(reference string, common Common, logger logrus.FieldLogger) (repo *remote.Repository, err error) {
	repo, err = remote.NewRepository(reference)
	if err != nil {
		if errors.Unwrap(err) == errdef.ErrInvalidReference {
			return nil, fmt.Errorf("%q: %v", reference, err)
		}
		return nil, err
	}
	registry := repo.Reference.Registry
	repo.PlainHTTP = opts.isPlainHttp(registry)
	repo.HandleWarning = opts.handleWarning(registry, logger)
	if repo.Client, err = opts.authClient(registry, common.Debug); err != nil {
		return nil, err
	}
	repo.SkipReferrersGC = true
	if opts.ReferrersAPI != nil {
		if err := repo.SetReferrersCapability(*opts.ReferrersAPI); err != nil {
			return nil, err
		}
	}
	return
}

// isPlainHttp returns the plain http flag for a given registry.
func (opts *Remote) isPlainHttp(registry string) bool {
	plainHTTP, enforced := opts.plainHTTP()
	if enforced {
		return plainHTTP
	}
	host, _, _ := net.SplitHostPort(registry)
	if host == "localhost" || registry == "localhost" {
		// not specified, defaults to plain http for localhost
		return true
	}
	return plainHTTP
}

// Modify modifies error during cmd execution.
func (opts *Remote) Modify(cmd *cobra.Command, err error) (error, bool) {
	var errResp *errcode.ErrorResponse

	if errors.Is(err, auth.ErrBasicCredentialNotFound) {
		return opts.DecorateCredentialError(err), true
	}

	if errors.As(err, &errResp) {
		cmd.SetErrPrefix(oerrors.RegistryErrorPrefix)
		return &oerrors.Error{
			Err: oerrors.TrimErrResp(err, errResp),
		}, true
	}
	return err, false
}

// DecorateCredentialError decorate error with recommendation.
func (opts *Remote) DecorateCredentialError(err error) *oerrors.Error {
	configPath := " "
	if path, pathErr := opts.ConfigPath(); pathErr == nil {
		configPath += fmt.Sprintf("at %q ", path)
	}
	return &oerrors.Error{
		Err:            oerrors.TrimErrBasicCredentialNotFound(err),
		Recommendation: fmt.Sprintf(`Please check whether the registry credential stored in the authentication file%sis correct`, configPath),
	}
}
