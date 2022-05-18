package option

import (
	ctls "crypto/tls"
	"crypto/x509"

	"github.com/spf13/pflag"
	"oras.land/oras/internal/http"
)

// TLS option struct.
type TLS struct {
	CACertFilePath string
	PlainHTTP      bool
	Insecure       bool
}

// ApplyFlags applies flags to a command flag set.
func (tls *TLS) ApplyFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&tls.Insecure, "insecure", "k", false, "allow connections to SSL registry without certs")
	fs.StringVarP(&tls.CACertFilePath, "ca-file", "", "", "server certificate authority file for the remote registry")
	fs.BoolVarP(&tls.PlainHTTP, "plain-http", "", false, "allow insecure connections to registry without SSL")
}

// Config assembles the tls config.
func (tls *TLS) Config() (config *ctls.Config, err error) {
	config = &ctls.Config{}
	var caPool *x509.CertPool
	if tls.CACertFilePath == "" {
		caPool = nil
	} else if caPool, err = http.LoadCertPool(tls.CACertFilePath); err != nil {
		return nil, err
	}

	config.RootCAs = caPool
	config.InsecureSkipVerify = tls.Insecure
	return
}
