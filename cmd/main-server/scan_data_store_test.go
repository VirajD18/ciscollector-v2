package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestStoreScanResultInsertsRow(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "main-server.sqlite")
	db, err := sql.Open("sqlite", sqliteDSN(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	app := testAppFromSQLiteDB(t, db, dbPath, "")
	now := time.Now().UTC()
	err = app.storeScanResult(ctx, &ScanDataRequest{
		SchemaVersion: "v1",
		Node: NodeInfo{
			ID:   "node-1",
			Name: "host-a",
			IP:   "127.0.0.1",
		},
		Report: map[string]interface{}{
			"Postgres Report": map[string]interface{}{
				"version": "16",
				"result":  []map[string]interface{}{{"Control": "1.1", "Status": "Pass"}},
			},
		},
		ScanRun: ScanRunMeta{
			Trigger:    "cron",
			TargetID:   "postgres:localhost:5432:hej",
			TargetHost: "localhost",
			TargetPort: "5432",
			TargetDB:   "hej",
			RunStatus:  "success",
			StartedAt:  now,
			FinishedAt: now,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scan_results`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("scan_results count=%d want 1", n)
	}
}
