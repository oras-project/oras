package option

import (
	"crypto/x509"

	"github.com/spf13/pflag"
	"oras.land/oras/internal/http"
)

type TLS struct {
	CACertFilePath string
	PlainHTTP      bool
	Insecure       bool
}

func (tls *TLS) ApplyFlagsTo(fs *pflag.FlagSet) {
	fs.BoolVarP(&tls.Insecure, "insecure", "k", false, "allow connections to SSL registry without certs")
	fs.StringVarP(&tls.CACertFilePath, "ca-file", "", "", "server certificate authority file for the remote registry")
	fs.BoolVarP(&tls.PlainHTTP, "plain-http", "", false, "allow insecure connections to registry without SSL")
}

func (tls *TLS) CertPool() (*x509.CertPool, error) {
	if tls.CACertFilePath == "" {
		return nil, nil
	}
	return http.LoadCertPool(tls.CACertFilePath)
}
