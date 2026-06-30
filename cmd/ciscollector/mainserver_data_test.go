package main

import (
	"testing"
	"time"

	"github.com/VirajD18/ciscollector-v2/pkg/config"
	"github.com/VirajD18/ciscollector-v2/pkg/mainserverclient"
	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
)

func TestPushScanDataToMainServerSkipsWhenDisabled(t *testing.T) {
	tests := []struct {
		name string
		cnf  *config.Config
	}{
		{
			name: "mainserver_disabled",
			cnf:  &config.Config{MainServer: config.MainServer{Enabled: false}},
		},
		{
			name: "nil_payload",
			cnf: &config.Config{
				MainServer: config.MainServer{Enabled: true, URL: "http://x", Token: "t"},
				App:        config.App{Hostname: "h"},
				Postgres:   &postgresdb.Postgres{},
			},
		},
	}
	client, err := mainserverclient.New(&config.Config{
		App:        config.App{Hostname: "h"},
		Postgres:   &postgresdb.Postgres{},
		MainServer: config.MainServer{Enabled: true, URL: "http://x", Token: "t"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	started := time.Now().UTC()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var fileData map[string]interface{}
			if tc.name == "nil_payload" {
				fileData = map[string]interface{}{"Password Manager Report": "x"}
			}
			pushScanDataToMainServer(tc.cnf, client, fileData, started, started, tc.cnf.Postgres, "cron", nil, "")
		})
	}
}
