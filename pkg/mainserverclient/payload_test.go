package mainserverclient

import (
	"testing"
	"time"

	"github.com/VirajD18/ciscollector-v2/model"
	"github.com/VirajD18/ciscollector-v2/pkg/config"
	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
)

func TestBuildNodePayload(t *testing.T) {
	client, err := New(&config.Config{
		App:        config.App{Hostname: "node-a"},
		Postgres:   &postgresdb.Postgres{Host: "10.0.0.1"},
		MainServer: config.MainServer{Enabled: true, URL: "http://localhost:8081", Token: "secret"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	started := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	finished := started.Add(2 * time.Second)

	tests := []struct {
		name      string
		cnf       *config.Config
		fileData  map[string]interface{}
		wantNil   bool
		wantCIS   bool
		wantHBA   bool
		wantToken string
	}{
		{
			name:     "empty",
			cnf:      &config.Config{MainServer: config.MainServer{URL: "http://x", Token: "t"}},
			fileData: map[string]interface{}{},
			wantNil:  true,
		},
		{
			name: "cis_only",
			cnf: &config.Config{
				MainServer: config.MainServer{URL: "http://localhost:8081", Token: "secret"},
				Postgres:   &postgresdb.Postgres{Host: "10.0.0.1"},
			},
			fileData: map[string]interface{}{
				"Postgres Report": map[string]interface{}{
					"version": "16",
					"result": []map[string]interface{}{
						{"Control": "1.1", "Status": "Pass"},
						{"Control": "2.1", "Status": "Fail"},
					},
				},
				"Users Report": []map[string]interface{}{{"title": "roles"}},
			},
			wantCIS:   true,
			wantToken: "secret",
		},
		{
			name: "hba_only",
			cnf: &config.Config{
				MainServer: config.MainServer{URL: "http://x", Token: "tok"},
				Postgres:   &postgresdb.Postgres{Host: "127.0.0.1"},
			},
			fileData: map[string]interface{}{
				"HBA Report": []map[string]interface{}{
					{"Title": "rule 1", "Control": 1, "Status": "Pass"},
				},
			},
			wantHBA: true,
		},
		{
			name: "text_report_skipped",
			cnf: &config.Config{
				MainServer: config.MainServer{URL: "http://x", Token: "tok"},
			},
			fileData: map[string]interface{}{
				"Postgres Report": "plain text report",
			},
			wantNil: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildNodePayload(tc.cnf, client, tc.fileData, started, finished)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("expected nil payload")
				}
				return
			}
			if got == nil {
				t.Fatalf("expected payload")
			}
			if got.SchemaVersion != "v1" {
				t.Fatalf("schema_version=%q", got.SchemaVersion)
			}
			if tc.wantCIS && got.Data.PostgresCIS == nil {
				t.Fatalf("expected postgres_cis")
			}
			if tc.wantHBA && got.Data.HBAScanResult == nil {
				t.Fatalf("expected hba_scan_result")
			}
			if tc.wantToken != "" && got.Node.AgentConfig.Server.Token != tc.wantToken {
				t.Fatalf("token=%q want %q", got.Node.AgentConfig.Server.Token, tc.wantToken)
			}
			if got.Data.PostgresCIS != nil {
				if got.Data.PostgresCIS.Version != "16" && tc.wantCIS {
					t.Fatalf("cis version=%q", got.Data.PostgresCIS.Version)
				}
				if got.Data.PostgresCIS.ScanMeta.DurationMs != 2000 {
					t.Fatalf("duration_ms=%d", got.Data.PostgresCIS.ScanMeta.DurationMs)
				}
			}
		})
	}
}

func TestCisSummaryFromResults(t *testing.T) {
	tests := []struct {
		name      string
		results   []*model.Result
		wantPass0 int
		wantFail0 int
	}{
		{
			name: "mixed",
			results: []*model.Result{
				{Control: "1.1", Status: "Pass"},
				{Control: "2.3", Status: "Fail"},
				{Control: "2.4", Status: "Pass"},
			},
			wantPass0: 2,
			wantFail0: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			summary := cisSummaryFromResults(tc.results)
			st, ok := summary[0].(*model.Status)
			if !ok {
				t.Fatalf("summary[0] type %T", summary[0])
			}
			if st.Pass != tc.wantPass0 || st.Fail != tc.wantFail0 {
				t.Fatalf("Pass=%d Fail=%d want %d/%d", st.Pass, st.Fail, tc.wantPass0, tc.wantFail0)
			}
		})
	}
}
