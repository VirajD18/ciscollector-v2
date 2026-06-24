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

const defaultServerDays = 825

type IssueServerOptions struct {
	CertDir   string
	SANs      []string
	ValidDays int
	Force     bool
}

func IssueServer(opts IssueServerOptions) error {
	dir := opts.CertDir
	if dir == "" {
		dir = DefaultCertDir
	}
	caPath := CAPath(dir)
	caKeyPath := CAKeyPath(dir)
	certPath := ServerCertPath(dir)
	keyPath := ServerKeyPath(dir)

	if !fileExists(caPath) || !fileExists(caKeyPath) {
		return fmt.Errorf("CA not found — run 'kshield cert init-ca' first (%s, %s)", caPath, caKeyPath)
	}
	if !opts.Force && (fileExists(certPath) || fileExists(keyPath)) {
		return fmt.Errorf("%s or %s already exists (use --force to overwrite)", certPath, keyPath)
	}

	sans := opts.SANs
	if len(sans) == 0 {
		sans = DefaultSANs()
	}
	entries, err := parseSANList(sans)
	if err != nil {
		return err
	}
	dnsNames, ipAddrs := sanEntriesToX509(entries)

	caCert, err := readPEMCert(caPath)
	if err != nil {
		return err
	}
	caKey, err := readPEMKey(caKeyPath)
	if err != nil {
		return err
	}

	if err := ensureCertDir(dir); err != nil {
		return err
	}

	days := opts.ValidDays
	if days <= 0 {
		days = defaultServerDays
	}

	serverKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("generate server key: %w", err)
	}

	cn := "KloudDB Shield Server"
	if len(dnsNames) > 0 {
		cn = dnsNames[0]
	} else if len(ipAddrs) > 0 {
		cn = ipAddrs[0].String()
	}

	now := time.Now().UTC()
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName:   cn,
			Organization: []string{"Klouddb"},
		},
		NotBefore:   now,
		NotAfter:    now.Add(time.Duration(days) * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    dnsNames,
		IPAddresses: ipAddrs,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("sign server cert: %w", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return fmt.Errorf("parse server cert: %w", err)
	}

	if err := writePEMKey(keyPath, serverKey); err != nil {
		return err
	}
	if err := writePEMCert(certPath, cert); err != nil {
		return err
	}
	return nil
}
