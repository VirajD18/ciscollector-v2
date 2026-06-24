package mainserverclient

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klouddb/klouddbshield/pkg/config"
	"github.com/klouddb/klouddbshield/pkg/postgresdb"
)

func TestHTTPSCollectorCommunication(t *testing.T) {
	const goodToken = "https-test-token"

	tests := []struct {
		name        string
		writeCAFile bool
		wantProbeOK bool
		wantPushOK  bool
	}{
		{
			name:        "tls_ca_file",
			writeCAFile: true,
			wantProbeOK: true,
			wantPushOK:  true,
		},
		{
			name:        "no_tls_trust_fails",
			wantProbeOK: false,
			wantPushOK:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/collector/config":
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{}`))
				case "/api/collector/heartbeat", "/api/collector/register":
					auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
					if auth != goodToken {
						http.Error(w, "unauthorized", http.StatusUnauthorized)
						return
					}
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"ok":true}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()

			ms := config.MainServer{
				Enabled: true,
				URL:     srv.URL,
				Token:   goodToken,
			}
			if tc.writeCAFile {
				ms.TLSCAFile = writeTestServerCA(t, srv)
			}

			cnf := &config.Config{
				App:        config.App{Hostname: "https-test-node"},
				Postgres:   &postgresdb.Postgres{},
				MainServer: ms,
			}

			client, err := New(cnf)
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			report := client.Probe(context.Background())
			if report.OK() != tc.wantProbeOK {
				t.Fatalf("Probe OK=%v want %v steps=%+v", report.OK(), tc.wantProbeOK, report.Steps)
			}

			err = client.PushHeartbeat(context.Background(), HeartbeatPayload{CronRunning: true})
			if (err == nil) != tc.wantPushOK {
				t.Fatalf("PushHeartbeat err=%v wantPushOK=%v", err, tc.wantPushOK)
			}
		})
	}
}

func writeTestServerCA(t *testing.T, srv *httptest.Server) string {
	t.Helper()

	conn, err := tls.Dial("tcp", srv.Listener.Addr().String(), &tls.Config{
		//nolint:gosec // test-only extraction of server certificate
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatalf("tls dial: %v", err)
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		t.Fatal("no peer certificates")
	}

	path := filepath.Join(t.TempDir(), "server-ca.pem")
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certs[0].Raw})
	if err := os.WriteFile(path, pemBytes, 0o644); err != nil {
		t.Fatalf("write ca file: %v", err)
	}
	return path
}

func TestNewHTTPClientTLSOptions(t *testing.T) {
	tests := []struct {
		name    string
		ms      config.MainServer
		wantErr bool
	}{
		{
			name: "http_url_ignores_tls_fields",
			ms:   config.MainServer{URL: "http://localhost:8081"},
		},
		{
			name:    "missing_ca_file",
			ms:      config.MainServer{URL: "https://localhost:8081", TLSCAFile: "/no/such/file.pem"},
			wantErr: true,
		},
		{
			name: "https_public_ca_uses_system_trust",
			ms:   config.MainServer{URL: "https://localhost:8081"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newHTTPClient(tc.ms)
			if (err != nil) != tc.wantErr {
				t.Fatalf("newHTTPClient() err=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestIsHTTPS(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{url: "https://localhost:8081", want: true},
		{url: "HTTPS://HOST:1", want: true},
		{url: "http://localhost:8081", want: false},
		{url: "", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			if got := isHTTPS(tc.url); got != tc.want {
				t.Fatalf("isHTTPS(%q)=%v want %v", tc.url, got, tc.want)
			}
		})
	}
}
