package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/VirajD18/ciscollector-v2/pkg/config"
	_ "modernc.org/sqlite"
)

func TestCollectorRegisterHandler(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "main-server.sqlite")
	db, err := sql.Open("sqlite", sqliteDSN(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	cfgPath := filepath.Join(dir, "kshieldconfig.toml")
	if err := os.WriteFile(cfgPath, []byte("[mainserver]\noffline_threshold_sec = 90\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	app := testAppFromSQLiteDB(t, db, dbPath, cfgPath)

	tests := []struct {
		name       string
		body       map[string]any
		wantStatus int
		wantRows   int
	}{
		{
			name: "registers_collector",
			body: map[string]any{
				"schema_version": "v1",
				"node_id":        "node-abc",
				"hostname":       "collector-1",
				"timestamp":      time.Now().UTC().Format(time.RFC3339),
				"schedule":       "*/2 * * * *",
				"scan_commands":  "postgres_cis",
				"scheduled_jobs": 1,
			},
			wantStatus: http.StatusOK,
			wantRows:   1,
		},
		{
			name: "rejects_bad_schema",
			body: map[string]any{
				"schema_version": "v0",
				"node_id":        "node-x",
				"hostname":       "x",
			},
			wantStatus: http.StatusBadRequest,
			wantRows:   1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			b, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/collector/register", bytes.NewReader(b))
			rec := httptest.NewRecorder()
			app.collectorRegisterHandler(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status %d want %d body=%s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			var count int
			if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM collector_status`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != tc.wantRows {
				t.Fatalf("collector_status rows %d want %d", count, tc.wantRows)
			}
		})
	}
}

func TestCollectorOfflineThresholdFromConfig(t *testing.T) {
	tests := []struct {
		name string
		body string
		want time.Duration
	}{
		{
			name: "configured_90",
			body: "[mainserver]\noffline_threshold_sec = 90\n",
			want: 90 * time.Second,
		},
		{
			name: "default_when_missing",
			body: "[mainserver]\nenabled = true\n",
			want: time.Duration(config.DefaultOfflineThresholdSec) * time.Second,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "kshieldconfig.toml")
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			app := &App{KshieldConfigPath: path}
			if got := app.collectorOfflineThreshold(); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
