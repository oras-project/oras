package http

import (
	"crypto/x509"
	"errors"
	"os"
)

func LoadRootCAs(path string) (*x509.CertPool, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if ok := pool.AppendCertsFromPEM(pemBytes); !ok {
		return nil, errors.New("Failed to add certificate authority in file: " + path)
	}
	return pool, nil
}
