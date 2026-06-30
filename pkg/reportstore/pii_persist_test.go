package reportstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
)

func TestPersistPIIReportUsesRunsTable(t *testing.T) {
	tests := []struct {
		name  string
		table string
	}{
		{name: "runs table", table: "runs"},
		{name: "scan_results table", table: "scan_results"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prev := RunsTable
			RunsTable = tc.table
			t.Cleanup(func() { RunsTable = prev })

			dir := t.TempDir()
			dbPath := filepath.Join(dir, "test.db")
			db, err := Open(dbPath)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			if tc.table == "scan_results" {
				if err := EnsureScanResultsSchema(context.Background(), db); err != nil {
					t.Fatal(err)
				}
			}

			pg := &postgresdb.Postgres{
				Host: "localhost", Port: "5432", User: "postgres", Password: "x", DBName: "hej",
			}
			cisData := map[string]interface{}{
				"Postgres Report": map[string]interface{}{"score": map[string]interface{}{"Pass": 5, "Fail": 1}},
			}

			if tc.table == "scan_results" {
				_, err = PersistScanResult(context.Background(), db, cisData, ScanResultMeta{
					RunMeta: RunMeta{
						Trigger: "cron", RunnerName: "test",
						Postgres: pg, StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(), RunStatus: "success",
					},
					NodeID: "node-1", Hostname: "test-host",
				})
			} else {
				_, err = Persist(context.Background(), db, cisData, RunMeta{
					Trigger: "cron", RunnerName: "test",
					Postgres: pg, StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(), RunStatus: "success",
				})
			}
			if err != nil {
				t.Fatal(err)
			}

			piiPayload := map[string]interface{}{
				"high_confidence": map[string]interface{}{
					"columns": []string{"table", "column", "label"},
					"rows":    [][]interface{}{{"users", "email", "Email"}},
				},
			}
			if err := PersistPIIReport(context.Background(), db, pg, piiPayload); err != nil {
				t.Fatal(err)
			}

			row, err := GetLatestRun(context.Background(), db, TargetID(pg))
			if err != nil || row == nil {
				t.Fatalf("get: %v", err)
			}
			if _, ok := row.Report["Postgres Report"]; !ok {
				t.Fatal("report_json should still contain Postgres Report")
			}
			if len(row.PiiReport) == 0 {
				t.Fatal("expected pii_report_json")
			}
		})
	}
}
