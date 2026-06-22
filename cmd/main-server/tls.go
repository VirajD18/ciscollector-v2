package main

import (
	"fmt"
	"os"
	"strings"
)

const (
	defaultTLSCert = "/etc/klouddbshield/certs/server.crt"
	defaultTLSKey  = "/etc/klouddbshield/certs/server.key"
)

type tlsConfig struct {
	enabled bool
	cert    string
	key     string
}

func resolveTLSConfig(certFlag, keyFlag string) (tlsConfig, error) {
	return resolveTLSConfigWithDefaults(certFlag, keyFlag, defaultTLSCert, defaultTLSKey)
}

func resolveTLSConfigWithDefaults(certFlag, keyFlag, defCert, defKey string) (tlsConfig, error) {
	cert := strings.TrimSpace(certFlag)
	key := strings.TrimSpace(keyFlag)

	switch {
	case cert == "" && key == "":
		if fileExists(defCert) && fileExists(defKey) {
			return tlsConfig{enabled: true, cert: defCert, key: defKey}, nil
		}
		return tlsConfig{}, nil
	case cert == "" || key == "":
		return tlsConfig{}, fmt.Errorf("both -tls-cert and -tls-key are required when either is set")
	}

	if !fileExists(cert) {
		return tlsConfig{}, fmt.Errorf("tls cert not found: %s", cert)
	}
	if !fileExists(key) {
		return tlsConfig{}, fmt.Errorf("tls key not found: %s", key)
	}
	return tlsConfig{enabled: true, cert: cert, key: key}, nil
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func defaultServerURL() string {
	if fileExists(defaultTLSCert) && fileExists(defaultTLSKey) {
		return "https://localhost:8081"
	}
	return "http://localhost:8081"
}
