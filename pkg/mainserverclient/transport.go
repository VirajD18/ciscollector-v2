package mainserverclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/VirajD18/ciscollector-v2/pkg/config"
)

func newHTTPClient(ms config.MainServer) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	if isHTTPS(ms.URL) {
		tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
		if ca := strings.TrimSpace(ms.TLSCAFile); ca != "" {
			pem, err := os.ReadFile(ca)
			if err != nil {
				return nil, fmt.Errorf("read mainserver.tls_ca_file: %w", err)
			}
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(pem) {
				return nil, fmt.Errorf("mainserver.tls_ca_file: no certificates found in %s", ca)
			}
			tlsCfg.RootCAs = pool
		}
		transport.TLSClientConfig = tlsCfg
	}

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}, nil
}

func isHTTPS(rawURL string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(rawURL)), "https://")
}
