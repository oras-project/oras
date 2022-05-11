package http

import (
	"crypto/x509"
	"errors"
	"os"
)

func LoadRootCAs(paths []string) (pool *x509.CertPool, err error) {
	pool, err = x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		pemBytes, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if ok := pool.AppendCertsFromPEM(pemBytes); !ok {
			return nil, errors.New("Failed to add certificate authority in file: " + path)
		}
	}
	return pool, nil
}
