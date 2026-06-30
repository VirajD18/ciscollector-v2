package mainserverclient

import (
	"testing"

	"github.com/VirajD18/ciscollector-v2/pkg/config"
	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
)

func TestNewRequiresEnabledMainServer(t *testing.T) {
	tests := []struct {
		name    string
		cnf     *config.Config
		wantErr bool
	}{
		{
			name:    "disabled",
			cnf:     &config.Config{MainServer: config.MainServer{Enabled: false}},
			wantErr: true,
		},
		{
			name: "missing_hostname",
			cnf: &config.Config{
				MainServer: config.MainServer{Enabled: true, URL: "http://x", Token: "t"},
			},
			wantErr: true,
		},
		{
			name: "ok",
			cnf: &config.Config{
				App:        config.App{Hostname: "node-a"},
				Postgres:   &postgresdb.Postgres{},
				MainServer: config.MainServer{Enabled: true, URL: "http://x", Token: "t"},
			},
			wantErr: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(tc.cnf)
			if (err != nil) != tc.wantErr {
				t.Fatalf("New() err=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}
