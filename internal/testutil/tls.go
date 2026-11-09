package testutil

import (
	"crypto/tls"
	"crypto/x509"
)

func tlsConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: false,
		RootCAs:            x509.NewCertPool(),
		MinVersion:         tls.VersionTLS12,
	}
}
