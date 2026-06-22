package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTLSConfig(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")
	if err := os.WriteFile(certPath, []byte("cert"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		certFlag    string
		keyFlag     string
		wantEnabled bool
		wantCert    string
		wantErr     bool
	}{
		{
			name:        "explicit_paths",
			certFlag:    certPath,
			keyFlag:     keyPath,
			wantEnabled: true,
			wantCert:    certPath,
		},
		{
			name:     "cert_only",
			certFlag: certPath,
			wantErr:  true,
		},
		{
			name:    "key_only",
			keyFlag: keyPath,
			wantErr: true,
		},
		{
			name:     "missing_cert_file",
			certFlag: filepath.Join(dir, "missing.crt"),
			keyFlag:  keyPath,
			wantErr:  true,
		},
		{
			name:     "missing_key_file",
			certFlag: certPath,
			keyFlag:  filepath.Join(dir, "missing.key"),
			wantErr:  true,
		},
		{
			name:        "no_flags_no_defaults",
			wantEnabled: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveTLSConfigWithDefaults(tc.certFlag, tc.keyFlag, "", "")
			if (err != nil) != tc.wantErr {
				t.Fatalf("resolveTLSConfig() err=%v wantErr=%v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if got.enabled != tc.wantEnabled {
				t.Fatalf("enabled=%v want %v", got.enabled, tc.wantEnabled)
			}
			if tc.wantCert != "" && got.cert != tc.wantCert {
				t.Fatalf("cert=%q want %q", got.cert, tc.wantCert)
			}
		})
	}
}

func TestResolveTLSConfigWithDefaultPaths(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")
	if err := os.WriteFile(certPath, []byte("cert"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := resolveTLSConfigWithDefaults("", "", certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if !got.enabled || got.cert != certPath || got.key != keyPath {
		t.Fatalf("got=%+v want enabled with default paths", got)
	}
}
