package testutil

import (
	"crypto/tls"
	"crypto/x509"
	"os"
)

func tlsConfig() *tls.Config {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
		// Fall back to bundled CA certs from the OS keychain if available.
		for _, path := range []string{
			"/etc/ssl/cert.pem",
			"/etc/ssl/certs/ca-certificates.crt",
			"/System/Library/Keychains/SystemRootCertificates.keychain",
		} {
			if b, e := os.ReadFile(path); e == nil {
				pool.AppendCertsFromPEM(b)
				break
			}
		}
	}
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    pool,
	}
}
