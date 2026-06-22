package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/klouddb/klouddbshield/pkg/reportstore"
	_ "modernc.org/sqlite"
)

func TestGucDriftFromSnapshots(t *testing.T) {
	tests := []struct {
		name            string
		baseline        map[string]string
		snapshots       map[string]map[string]string
		wantHosts       int
		wantMatched     int
		wantDrifting    int
		wantMissing     int
		wantDriftRows   int
		wantMissingRows int
	}{
		{
			name:     "no baseline",
			baseline: nil,
			snapshots: map[string]map[string]string{
				"postgres:host:5432:db": {"ssl": "on"},
			},
			wantHosts: 0,
		},
		{
			name:     "all matched",
			baseline: map[string]string{"ssl": "on", "shared_buffers": "128MB"},
			snapshots: map[string]map[string]string{
				"postgres:host:5432:db": {"ssl": "ON", "shared_buffers": "128mb"},
			},
			wantHosts:    1,
			wantMatched:  1,
			wantDrifting: 0,
		},
		{
			name:     "drift and missing",
			baseline: map[string]string{"ssl": "on", "max_connections": "200"},
			snapshots: map[string]map[string]string{
				"postgres:a:5432:db": {"ssl": "off"},
				"postgres:b:5432:db": {"ssl": "on", "max_connections": "200"},
			},
			wantHosts:       2,
			wantMatched:     1,
			wantDrifting:    1,
			wantMissing:     1,
			wantDriftRows:   1,
			wantMissingRows: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := openTestGucDB(t)
			ctx := context.Background()
			if tt.baseline != nil {
				if err := reportstore.UpsertGucBaseline(ctx, db, "test", tt.baseline); err != nil {
					t.Fatal(err)
				}
			}
			for targetID, settings := range tt.snapshots {
				if err := reportstore.UpsertServerGucSnapshot(ctx, db, targetID, targetID, "node-1", settings); err != nil {
					t.Fatal(err)
				}
			}

			resp, err := NewSQLiteService(db).GucDrift(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if resp.Stats.HostsCompared != tt.wantHosts {
				t.Fatalf("hosts_compared=%d want %d", resp.Stats.HostsCompared, tt.wantHosts)
			}
			if resp.Stats.MatchedServers != tt.wantMatched {
				t.Fatalf("matched=%d want %d", resp.Stats.MatchedServers, tt.wantMatched)
			}
			if resp.Stats.DriftingServers != tt.wantDrifting {
				t.Fatalf("drifting=%d want %d", resp.Stats.DriftingServers, tt.wantDrifting)
			}
			if resp.Stats.MissingServers != tt.wantMissing {
				t.Fatalf("missing_servers=%d want %d", resp.Stats.MissingServers, tt.wantMissing)
			}
			driftRows, missingRows := 0, 0
			for _, row := range resp.Rows {
				switch row.Status {
				case "drift":
					driftRows++
				case "missing":
					missingRows++
				}
			}
			if driftRows != tt.wantDriftRows {
				t.Fatalf("drift_rows=%d want %d", driftRows, tt.wantDriftRows)
			}
			if missingRows != tt.wantMissingRows {
				t.Fatalf("missing_rows=%d want %d", missingRows, tt.wantMissingRows)
			}
		})
	}
}

func openTestGucDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := reportstore.EnsureScanResultsSchema(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return db
}
