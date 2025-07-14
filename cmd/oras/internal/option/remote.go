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

const (
	caFileFlag                 = "ca-file"
	certFileFlag               = "cert-file"
	keyFileFlag                = "key-file"
	usernameFlag               = "username"
	passwordFlag               = "password"
	passwordFromStdinFlag      = "password-stdin"
	identityTokenFlag          = "identity-token"
	identityTokenFromStdinFlag = "identity-token-stdin"
)

// Remote options struct contains flags and arguments specifying one registry.
// Remote implements oerrors.Handler and interface.
type Remote struct {
	DistributionSpec
	CACertFilePath  string
	CertFilePath    string
	KeyFilePath     string
	Insecure        bool
	Configs         []string
	Username        string
	secretFromStdin bool
	Secret          string
	flagPrefix      string

	resolveFlag           []string
	applyDistributionSpec bool
	headerFlags           []string
	headers               http.Header
	warned                map[string]*sync.Map
	plainHTTP             func() (plainHTTP bool, enforced bool)
	store                 credentials.Store
}

// EnableDistributionSpecFlag set distribution specification flag as applicable.
func (remo *Remote) EnableDistributionSpecFlag() {
	remo.applyDistributionSpec = true
}

// ApplyFlags applies flags to a command flag set.
func (remo *Remote) ApplyFlags(fs *pflag.FlagSet) {
	remo.ApplyFlagsWithPrefix(fs, "", "")
	remo.applyStdinFlags(fs)
}

func (remo *Remote) applyStdinFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&remo.secretFromStdin, passwordFromStdinFlag, false, "read password from stdin")
	fs.BoolVar(&remo.secretFromStdin, identityTokenFromStdinFlag, false, "read identity token from stdin")
}

// ApplyFlagsWithPrefix applies flags to a command flag set with a prefix string.
// Commonly used for non-unary remote targets.
func (remo *Remote) ApplyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	var (
		shortUser     string
		shortPassword string
		shortHeader   string
	)
	if prefix == "" {
		shortUser, shortPassword = "u", "p"
		shortHeader = "H"
	}
	remo.flagPrefix = prefix

	if remo.applyDistributionSpec {
		remo.DistributionSpec.ApplyFlagsWithPrefix(fs, prefix, description)
	}
	fs.StringVarP(&remo.Username, remo.flagPrefix+usernameFlag, shortUser, "", description+"registry username")
	fs.StringVarP(&remo.Secret, remo.flagPrefix+passwordFlag, shortPassword, "", description+"registry password or identity token")
	fs.StringVar(&remo.Secret, remo.flagPrefix+identityTokenFlag, "", description+"registry identity token")
	fs.BoolVar(&remo.Insecure, remo.flagPrefix+"insecure", false, "allow connections to "+description+"SSL registry without certs")
	plainHTTPFlagName := remo.flagPrefix + "plain-http"
	plainHTTP := fs.Bool(plainHTTPFlagName, false, "allow insecure connections to "+description+"registry without SSL check")
	remo.plainHTTP = func() (bool, bool) {
		return *plainHTTP, fs.Changed(plainHTTPFlagName)
	}
	fs.StringVar(&remo.CACertFilePath, remo.flagPrefix+caFileFlag, "", "server certificate authority file for the remote "+description+"registry")
	fs.StringVarP(&remo.CertFilePath, remo.flagPrefix+certFileFlag, "", "", "client certificate file for the remote "+description+"registry")
	fs.StringVarP(&remo.KeyFilePath, remo.flagPrefix+keyFileFlag, "", "", "client private key file for the remote "+description+"registry")
	fs.StringArrayVar(&remo.resolveFlag, remo.flagPrefix+"resolve", nil, "customized DNS for "+description+"registry, formatted in `host:port:address[:address_port]`")
	fs.StringArrayVar(&remo.Configs, remo.flagPrefix+"registry-config", nil, "`path` of the authentication file for "+description+"registry")
	fs.StringArrayVarP(&remo.headerFlags, remo.flagPrefix+"header", shortHeader, nil, "add custom headers to "+description+"requests")
}

// CheckStdinConflict checks if PasswordFromStdin or IdentityTokenFromStdin of a
// *pflag.FlagSet conflicts with read file from input.
func CheckStdinConflict(flags *pflag.FlagSet) error {
	switch {
	case flags.Changed(passwordFromStdinFlag):
		return fmt.Errorf("`-` read file from input and `--%s` read password from input cannot be both used", passwordFromStdinFlag)
	case flags.Changed(identityTokenFromStdinFlag):
		return fmt.Errorf("`-` read file from input and `--%s` read identity token from input cannot be both used", identityTokenFromStdinFlag)
	}
	return nil
}

// Parse tries to read password with optional cmd prompt.
func (remo *Remote) Parse(cmd *cobra.Command) error {
	usernameAndIdTokenFlags := []string{remo.flagPrefix + usernameFlag, remo.flagPrefix + identityTokenFlag}
	passwordAndIdTokenFlags := []string{remo.flagPrefix + passwordFlag, remo.flagPrefix + identityTokenFlag}
	certFileAndKeyFileFlags := []string{remo.flagPrefix + certFileFlag, remo.flagPrefix + keyFileFlag}
	if cmd.Flags().Lookup(identityTokenFromStdinFlag) != nil {
		usernameAndIdTokenFlags = append(usernameAndIdTokenFlags, identityTokenFromStdinFlag)
		passwordAndIdTokenFlags = append(passwordAndIdTokenFlags, identityTokenFromStdinFlag)
	}
	if cmd.Flags().Lookup(passwordFromStdinFlag) != nil {
		passwordAndIdTokenFlags = append(passwordAndIdTokenFlags, passwordFromStdinFlag)
	}
	if err := oerrors.CheckMutuallyExclusiveFlags(cmd.Flags(), usernameAndIdTokenFlags...); err != nil {
		return err
	}
	if err := oerrors.CheckMutuallyExclusiveFlags(cmd.Flags(), passwordAndIdTokenFlags...); err != nil {
		return err
	}
	if err := remo.parseCustomHeaders(); err != nil {
		return err
	}
	if err := oerrors.CheckRequiredTogetherFlags(cmd.Flags(), certFileAndKeyFileFlags...); err != nil {
		return err
	}
	return remo.readSecret(cmd)
}

// readSecret tries to read password or identity token with
// optional cmd prompt.
func (remo *Remote) readSecret(cmd *cobra.Command) (err error) {
	if cmd.Flags().Changed(identityTokenFlag) {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "WARNING! Using --identity-token via the CLI is insecure. Use --identity-token-stdin.")
	} else if cmd.Flags().Changed(passwordFlag) {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "WARNING! Using --password via the CLI is insecure. Use --password-stdin.")
	} else if remo.secretFromStdin {
		// Prompt for credential
		secret, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		remo.Secret = strings.TrimSuffix(string(secret), "\n")
		remo.Secret = strings.TrimSuffix(remo.Secret, "\r")
	}
	return nil
}

// parseResolve parses resolve flag.
func (remo *Remote) parseResolve(baseDial onet.DialFunc) (onet.DialFunc, error) {
	if len(remo.resolveFlag) == 0 {
		return baseDial, nil
	}

	formatError := func(param, message string) error {
		return fmt.Errorf("failed to parse resolve flag %q: %s", param, message)
	}
	var dialer onet.Dialer
	for _, r := range remo.resolveFlag {
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
func (remo *Remote) tlsConfig() (*tls.Config, error) {
	config := &tls.Config{
		InsecureSkipVerify: remo.Insecure,
	}
	if remo.CACertFilePath != "" {
		var err error
		config.RootCAs, err = crypto.LoadCertPool(remo.CACertFilePath)
		if err != nil {
			return nil, err
		}
	}
	if remo.CertFilePath != "" && remo.KeyFilePath != "" {
		cert, err := tls.LoadX509KeyPair(remo.CertFilePath, remo.KeyFilePath)
		if err != nil {
			return nil, err
		}
		config.Certificates = []tls.Certificate{cert}
	}
	return config, nil
}

// authClient assembles a oras auth client.
func (remo *Remote) authClient(registry string, debug bool) (client *auth.Client, err error) {
	config, err := remo.tlsConfig()
	if err != nil {
		return nil, err
	}
	baseTransport := http.DefaultTransport.(*http.Transport).Clone()
	baseTransport.TLSClientConfig = config
	dialContext, err := remo.parseResolve(baseTransport.DialContext)
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
		Header: remo.headers,
	}
	client.SetUserAgent("oras/" + version.GetVersion())
	if debug {
		client.Client.Transport = trace.NewTransport(client.Client.Transport)
	}

	cred := remo.Credential()
	if cred != auth.EmptyCredential {
		client.Credential = func(ctx context.Context, s string) (auth.Credential, error) {
			return cred, nil
		}
	} else {
		var err error
		remo.store, err = credential.NewStore(remo.Configs...)
		if err != nil {
			return nil, err
		}
		client.Credential = credentials.Credential(remo.store)
	}
	return
}

// ConfigPath returns the config path of the credential store.
func (remo *Remote) ConfigPath() (string, error) {
	if remo.store == nil {
		return "", errors.New("no credential store initialized")
	}
	if ds, ok := remo.store.(*credentials.DynamicStore); ok {
		return ds.ConfigPath(), nil
	}
	return "", errors.New("store doesn't support getting config path")
}

func (remo *Remote) parseCustomHeaders() error {
	if len(remo.headerFlags) != 0 {
		headers := map[string][]string{}
		for _, h := range remo.headerFlags {
			name, value, found := strings.Cut(h, ":")
			if !found || strings.TrimSpace(name) == "" {
				// In conformance to the RFC 2616 specification
				// Reference: https://www.rfc-editor.org/rfc/rfc2616#section-4.2
				return fmt.Errorf("invalid header: %q", h)
			}
			headers[name] = append(headers[name], value)
		}
		remo.headers = headers
	}
	return nil
}

// Credential returns a credential based on the remote options.
func (remo *Remote) Credential() auth.Credential {
	return credential.Credential(remo.Username, remo.Secret)
}

func (remo *Remote) handleWarning(registry string, logger logrus.FieldLogger) func(warning remote.Warning) {
	if remo.warned == nil {
		remo.warned = make(map[string]*sync.Map)
	}
	warned := remo.warned[registry]
	if warned == nil {
		warned = &sync.Map{}
		remo.warned[registry] = warned
	}
	logger = logger.WithField("registry", registry)
	return func(warning remote.Warning) {
		if _, loaded := warned.LoadOrStore(warning.WarningValue, struct{}{}); !loaded {
			logger.Warn(warning.Text)
		}
	}
}

// NewRegistry assembles a oras remote registry.
func (remo *Remote) NewRegistry(registry string, common Common, logger logrus.FieldLogger) (reg *remote.Registry, err error) {
	reg, err = remote.NewRegistry(registry)
	if err != nil {
		return nil, err
	}
	registry = reg.Reference.Registry
	reg.PlainHTTP = remo.isPlainHttp(registry)
	reg.HandleWarning = remo.handleWarning(registry, logger)
	if reg.Client, err = remo.authClient(registry, common.Debug); err != nil {
		return nil, err
	}
	return
}

// NewRepository assembles a oras remote repository.
func (remo *Remote) NewRepository(reference string, common Common, logger logrus.FieldLogger) (repo *remote.Repository, err error) {
	repo, err = remote.NewRepository(reference)
	if err != nil {
		if errors.Unwrap(err) == errdef.ErrInvalidReference {
			return nil, fmt.Errorf("%q: %v", reference, err)
		}
		return nil, err
	}
	registry := repo.Reference.Registry
	repo.PlainHTTP = remo.isPlainHttp(registry)
	repo.HandleWarning = remo.handleWarning(registry, logger)
	if repo.Client, err = remo.authClient(registry, common.Debug); err != nil {
		return nil, err
	}
	repo.SkipReferrersGC = true
	if remo.ReferrersAPI != nil {
		if err := repo.SetReferrersCapability(*remo.ReferrersAPI); err != nil {
			return nil, err
		}
	}
	return
}

// isPlainHttp returns the plain http flag for a given registry.
func (remo *Remote) isPlainHttp(registry string) bool {
	plainHTTP, enforced := remo.plainHTTP()
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

// ModifyError modifies error during cmd execution.
func (remo *Remote) ModifyError(cmd *cobra.Command, err error) (error, bool) {
	if errors.Is(err, auth.ErrBasicCredentialNotFound) {
		return remo.DecorateCredentialError(err), true
	}

	var errResp *errcode.ErrorResponse
	if errors.As(err, &errResp) {
		cmd.SetErrPrefix(oerrors.RegistryErrorPrefix)
		return &oerrors.Error{
			Err: oerrors.ReportErrResp(errResp),
		}, true
	}
	return err, false
}

// DecorateCredentialError decorate error with recommendation.
func (remo *Remote) DecorateCredentialError(err error) *oerrors.Error {
	configPath := " "
	if path, pathErr := remo.ConfigPath(); pathErr == nil {
		configPath += fmt.Sprintf("at %q ", path)
	}
	return &oerrors.Error{
		Err:            oerrors.TrimErrBasicCredentialNotFound(err),
		Recommendation: fmt.Sprintf(`Please check whether the registry credential stored in the authentication file%sis correct`, configPath),
	}
}
