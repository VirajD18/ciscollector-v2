package service

import (
	"context"
	"testing"
	"time"

	"github.com/klouddb/klouddbshield/pkg/postgresdb"
	"github.com/klouddb/klouddbshield/pkg/reportstore"
)

func TestPiiScannerStatusMessages(t *testing.T) {
	ctx := context.Background()
	pg := &postgresdb.Postgres{Host: "localhost", Port: "5432", DBName: "hej"}
	started := time.Now().UTC()

	tests := []struct {
		name        string
		piiReport   map[string]interface{}
		wantAvail   bool
		wantStatus  string
		wantMessage string
		wantRows    int
	}{
		{
			name: "no tables",
			piiReport: map[string]interface{}{
				"status":  "no_tables",
				"message": "No tables found in database matching the PII scan criteria.",
				"schema":  "public",
			},
			wantAvail:   true,
			wantStatus:  "no_tables",
			wantMessage: "No tables found in database matching the PII scan criteria.",
		},
		{
			name: "no data",
			piiReport: map[string]interface{}{
				"status":  "no_data",
				"message": "No PII data found in database.",
				"schema":  "public",
			},
			wantAvail:   true,
			wantStatus:  "no_data",
			wantMessage: "No PII data found in database.",
		},
		{
			name: "scan error",
			piiReport: map[string]interface{}{
				"status":  "error",
				"message": "PII scan failed.",
				"error":   "relation missing_table does not exist",
			},
			wantAvail:   true,
			wantStatus:  "error",
			wantMessage: "PII scan failed.: relation missing_table does not exist",
		},
		{
			name: "success with findings",
			piiReport: map[string]interface{}{
				"status": "success",
				"high_confidence": map[string]interface{}{
					"columns": []interface{}{"table", "column", "label", "confidence", "detector", "matched"},
					"rows": []interface{}{
						[]interface{}{"employees", "email", "Email", "High", "regex", "10/10"},
					},
				},
				"schema": "public",
			},
			wantAvail:  true,
			wantStatus: "success",
			wantRows:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := OpenTestSQLiteDB(t)
			defer db.Close()
			reportstore.RunsTable = "scan_results"
			if err := reportstore.PersistPIIReport(ctx, db, pg, tt.piiReport); err != nil {
				t.Fatal(err)
			}
			_, err := reportstore.PersistScanResult(ctx, db, map[string]interface{}{
				"Postgres Report": map[string]interface{}{"version": "16", "result": []interface{}{}},
			}, reportstore.ScanResultMeta{
				RunMeta: reportstore.RunMeta{
					Trigger: "manual", RunnerName: "ciscollector", Postgres: pg,
					StartedAt: started, FinishedAt: started, RunStatus: "success",
				},
				NodeID: "node-1", Hostname: "localhost",
			})
			if err != nil {
				t.Fatal(err)
			}

			resp, err := NewSQLiteService(db).PiiScanner(ctx, "localhost:5432")
			if err != nil {
				t.Fatal(err)
			}
			if resp.Available != tt.wantAvail {
				t.Fatalf("available=%v want %v", resp.Available, tt.wantAvail)
			}
			if resp.Status != tt.wantStatus {
				t.Fatalf("status=%q want %q", resp.Status, tt.wantStatus)
			}
			if tt.wantMessage != "" && resp.Message != tt.wantMessage {
				t.Fatalf("message=%q want %q", resp.Message, tt.wantMessage)
			}
			if len(resp.Rows) != tt.wantRows {
				t.Fatalf("rows=%d want %d", len(resp.Rows), tt.wantRows)
			}
		})
	}
}

func TestDecodePIIStatus(t *testing.T) {
	tests := []struct {
		name      string
		report    map[string]interface{}
		wantStat  string
		wantMsg   string
		wantError string
	}{
		{
			name: "top level",
			report: map[string]interface{}{
				"status": "no_tables", "message": "No tables found", "error": "detail",
			},
			wantStat: "no_tables", wantMsg: "No tables found", wantError: "detail",
		},
		{
			name:     "empty",
			report:   map[string]interface{}{},
			wantStat: "", wantMsg: "", wantError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, msg, errText := decodePIIStatus(tt.report)
			if st != tt.wantStat || msg != tt.wantMsg || errText != tt.wantError {
				t.Fatalf("got (%q,%q,%q) want (%q,%q,%q)", st, msg, errText, tt.wantStat, tt.wantMsg, tt.wantError)
			}
		})
	}
}
