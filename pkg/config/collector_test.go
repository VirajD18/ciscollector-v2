package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klouddb/klouddbshield/pkg/postgresdb"
)

func TestMainServerValidate(t *testing.T) {
	tests := []struct {
		name    string
		ms      MainServer
		wantErr bool
	}{
		{name: "disabled_ok", ms: MainServer{Enabled: false}},
		{name: "enabled_missing_url", ms: MainServer{Enabled: true, Token: "x"}, wantErr: true},
		{name: "enabled_ok", ms: MainServer{Enabled: true, URL: "http://h:1", Token: "x"}},
		{name: "enabled_https_ok", ms: MainServer{Enabled: true, URL: "https://h:1", Token: "x"}},
		{name: "enabled_missing_tls_ca_file", ms: MainServer{Enabled: true, URL: "https://h:1", Token: "x", TLSCAFile: "/no/such/ca.pem"}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.ms.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() err=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestMainServerValidateTLS_CAFileReadable(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(caPath, []byte("-----BEGIN CERTIFICATE-----\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		ms      MainServer
		wantErr bool
	}{
		{
			name: "tls_ca_file_ok",
			ms: MainServer{
				Enabled:   true,
				URL:       "https://shield.example.com:8081",
				Token:     "secret",
				TLSCAFile: caPath,
			},
		},
		{
			name: "tls_ca_file_missing",
			ms: MainServer{
				Enabled:   true,
				URL:       "https://shield.example.com:8081",
				Token:     "secret",
				TLSCAFile: filepath.Join(dir, "missing.pem"),
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.ms.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() err=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestCollectorScheduleReady(t *testing.T) {
	tests := []struct {
		name string
		c    *Config
		want bool
	}{
		{
			name: "ready",
			c: &Config{
				Postgres:  &postgresdb.Postgres{},
				Collector: Collector{Schedule: "0 0 * * *", ScanCommands: "postgres_cis"},
			},
			want: true,
		},
		{name: "missing_schedule", c: &Config{Postgres: &postgresdb.Postgres{}, Collector: Collector{ScanCommands: "x"}}, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.c.CollectorScheduleReady(); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
