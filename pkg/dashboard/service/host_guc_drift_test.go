package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/VirajD18/ciscollector-v2/pkg/reportstore"
)

func TestBuildHostGucDriftView(t *testing.T) {
	tests := []struct {
		name         string
		baseline     map[string]string
		snapshotHost string
		targetID     string
		live         map[string]string
		wantStatus   string
		wantRows     int
		wantMissing  int
	}{
		{
			name:       "no baseline",
			live:       map[string]string{"ssl": "on"},
			targetID:   "postgres:h:5432:db",
			wantStatus: "no_baseline",
		},
		{
			name:         "matched",
			baseline:     map[string]string{"ssl": "on", "max_connections": "100"},
			snapshotHost: "collector-a",
			targetID:     "postgres:h:5432:db",
			live:         map[string]string{"ssl": "on", "max_connections": "100"},
			wantStatus:   "matched",
		},
		{
			name:         "drift and missing",
			baseline:     map[string]string{"ssl": "on", "max_connections": "200", "missing_guc": "x"},
			snapshotHost: "collector-a",
			targetID:     "postgres:h:5432:db",
			live:         map[string]string{"ssl": "off", "max_connections": "200"},
			wantStatus:   "drifted",
			wantRows:     2,
			wantMissing:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := openHostGucTestDB(t)
			ctx := context.Background()
			if tt.baseline != nil {
				if err := reportstore.UpsertGucBaseline(ctx, db, "global", tt.baseline); err != nil {
					t.Fatal(err)
				}
			}
			if tt.live != nil && tt.targetID != "" {
				host := tt.snapshotHost
				if host == "" {
					host = tt.targetID
				}
				if err := reportstore.UpsertServerGucSnapshot(ctx, db, tt.targetID, host, "node-1", tt.live); err != nil {
					t.Fatal(err)
				}
			}

			view := NewSQLiteService(db).buildHostGucDriftView(ctx, tt.snapshotHost, tt.targetID)
			if view.Status != tt.wantStatus {
				t.Fatalf("status=%q want %q", view.Status, tt.wantStatus)
			}
			if len(view.Rows) != tt.wantRows {
				t.Fatalf("rows=%d want %d", len(view.Rows), tt.wantRows)
			}
			if view.MissingCount != tt.wantMissing {
				t.Fatalf("missing_count=%d want %d", view.MissingCount, tt.wantMissing)
			}
		})
	}
}

func openHostGucTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := OpenTestSQLiteDB(t)
	t.Cleanup(func() { _ = db.Close() })
	return db
}
