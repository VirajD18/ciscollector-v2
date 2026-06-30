package mainserverclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VirajD18/ciscollector-v2/pkg/config"
	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
)

func TestProbe(t *testing.T) {
	const goodToken = "test-token"

	tests := []struct {
		name       string
		token      string
		registerOK bool
		wantOK     bool
		wantSteps  int
		failStep   string
	}{
		{
			name:       "all_ok",
			token:      goodToken,
			registerOK: true,
			wantOK:     true,
			wantSteps:  3,
		},
		{
			name:       "bad_token",
			token:      "wrong",
			registerOK: true,
			wantOK:     false,
			wantSteps:  3,
			failStep:   "auth (heartbeat)",
		},
		{
			name:       "register_missing",
			token:      goodToken,
			registerOK: false,
			wantOK:     false,
			wantSteps:  3,
			failStep:   "register",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
					if r.URL.Path == "/api/collector/register" && !tc.registerOK {
						http.NotFound(w, r)
						return
					}
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"ok":true}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()

			cnf := &config.Config{
				App:        config.App{Hostname: "probe-node"},
				Postgres:   &postgresdb.Postgres{},
				MainServer: config.MainServer{Enabled: true, URL: srv.URL, Token: tc.token},
			}
			client, err := New(cnf)
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			report := client.Probe(context.Background())
			if len(report.Steps) != tc.wantSteps {
				t.Fatalf("steps=%d want %d", len(report.Steps), tc.wantSteps)
			}
			if report.OK() != tc.wantOK {
				t.Fatalf("OK()=%v want %v steps=%+v", report.OK(), tc.wantOK, report.Steps)
			}
			if tc.failStep != "" {
				for _, s := range report.Steps {
					if s.Name == tc.failStep && s.OK {
						t.Fatalf("expected step %q to fail", tc.failStep)
					}
				}
			}
		})
	}
}
