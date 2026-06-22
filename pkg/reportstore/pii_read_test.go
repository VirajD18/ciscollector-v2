package reportstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/klouddb/klouddbshield/pkg/postgresdb"
)

func TestGetLatestRunWithPII(t *testing.T) {
	tests := []struct {
		name      string
		piiFirst  bool
		wantPII   bool
	}{
		{
			name:     "returns row with pii when newer cron row has no pii",
			piiFirst: true,
			wantPII:  true,
		},
		{
			name:     "no pii rows",
			piiFirst: false,
			wantPII:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prev := RunsTable
			RunsTable = "scan_results"
			t.Cleanup(func() { RunsTable = prev })

			dir := t.TempDir()
			db, err := Open(filepath.Join(dir, "test.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := EnsureScanResultsSchema(context.Background(), db); err != nil {
				t.Fatal(err)
			}

			pg := &postgresdb.Postgres{Host: "localhost", Port: "5432", DBName: "hej"}
			tid := TargetID(pg)
			meta := ScanResultMeta{
				RunMeta: RunMeta{
					Trigger: "cron", RunnerName: "test", Postgres: pg,
					StartedAt: time.Now().UTC().Add(-2 * time.Minute), FinishedAt: time.Now().UTC(),
					RunStatus: "success",
				},
				NodeID: "n1", Hostname: "host1",
			}

			if tc.piiFirst {
				_, err = PersistScanResult(context.Background(), db, map[string]interface{}{"Postgres Report": map[string]interface{}{}}, meta)
				if err != nil {
					t.Fatal(err)
				}
				if err := PersistPIIReport(context.Background(), db, pg, map[string]interface{}{
					"high_confidence": map[string]interface{}{
						"columns": []string{"table", "column", "label"},
						"rows":    [][]interface{}{{"users", "email", "Email"}},
					},
				}); err != nil {
					t.Fatal(err)
				}
				meta.StartedAt = time.Now().UTC()
				meta.FinishedAt = time.Now().UTC()
			}

			_, err = PersistScanResult(context.Background(), db, map[string]interface{}{"HBA Report": []interface{}{}}, meta)
			if err != nil {
				t.Fatal(err)
			}

			if !tc.piiFirst {
				latest, err := GetLatestRunWithPII(context.Background(), db, tid)
				if err != nil {
					t.Fatal(err)
				}
				if latest != nil {
					t.Fatal("expected no pii row")
				}
				return
			}

			got, err := GetLatestRunWithPII(context.Background(), db, tid)
			if err != nil {
				t.Fatal(err)
			}
			if got == nil || len(got.PiiReport) == 0 {
				t.Fatal("expected pii row")
			}
			latest, err := GetLatestRun(context.Background(), db, tid)
			if err != nil || latest == nil {
				t.Fatal(err)
			}
			if len(latest.PiiReport) != 0 {
				t.Fatal("latest run should not have pii; cron row is newer")
			}
		})
	}
}
