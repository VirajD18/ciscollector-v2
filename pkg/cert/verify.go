package cert

import (
	"crypto/x509"
	"fmt"
)

// VerifyServer checks server.crt is signed by ca.crt.
func VerifyServer(certDir string) error {
	dir := certDir
	if dir == "" {
		dir = DefaultCertDir
	}
	caCert, err := readPEMCert(CAPath(dir))
	if err != nil {
		return err
	}
	serverCert, err := readPEMCert(ServerCertPath(dir))
	if err != nil {
		return err
	}

	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	if _, err := serverCert.Verify(x509.VerifyOptions{
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}); err != nil {
		return fmt.Errorf("server certificate verification failed: %w", err)
	}
	return nil
}
