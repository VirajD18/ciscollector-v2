package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"
)

const defaultCADays = 3650

type InitCAOptions struct {
	CertDir   string
	Force     bool
	ValidDays int
}

func InitCA(opts InitCAOptions) error {
	dir := opts.CertDir
	if dir == "" {
		dir = DefaultCertDir
	}
	caPath := CAPath(dir)
	keyPath := CAKeyPath(dir)

	if !opts.Force && (fileExists(caPath) || fileExists(keyPath)) {
		return fmt.Errorf("%s or %s already exists (use --force to overwrite)", caPath, keyPath)
	}
	if err := ensureCertDir(dir); err != nil {
		return err
	}

	days := opts.ValidDays
	if days <= 0 {
		days = defaultCADays
	}

	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("generate ca key: %w", err)
	}

	now := time.Now().UTC()
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName:   "KloudDB Shield CA",
			Organization: []string{"Klouddb"},
		},
		NotBefore:             now,
		NotAfter:              now.Add(time.Duration(days) * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("create ca cert: %w", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return fmt.Errorf("parse ca cert: %w", err)
	}

	if err := writePEMKey(keyPath, key); err != nil {
		return err
	}
	if err := writePEMCert(caPath, cert); err != nil {
		return err
	}
	return nil
}
