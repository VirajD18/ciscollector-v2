package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
)

func TestCollectorNodeLogsHandler(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "main-server.sqlite")
	db, err := sql.Open("sqlite", sqliteDSN(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	app := testAppFromSQLiteDB(t, db, dbPath, "")
	ctx := context.Background()
	if err := app.Svc.RegisterCollector(ctx, "node-logs-1", "host-a", "", time.Now().UTC(), 1, "setup"); err != nil {
		t.Fatal(err)
	}
	if err := app.Svc.RecordCollectorLog(ctx, "node-logs-1", "error", "postgres connection refused", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := app.Svc.RecordCollectorLog(ctx, "node-logs-1", "info", "cron tick started", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		nodeID     string
		wantStatus int
		wantLogs   int
	}{
		{
			name:       "returns_logs",
			nodeID:     "node-logs-1",
			wantStatus: http.StatusOK,
			wantLogs:   2,
		},
		{
			name:       "empty_for_unknown_node",
			nodeID:     "missing-node",
			wantStatus: http.StatusOK,
			wantLogs:   0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/collector/nodes/"+tc.nodeID+"/logs", nil)
			req = mux.SetURLVars(req, map[string]string{"id": tc.nodeID})
			rec := httptest.NewRecorder()
			app.collectorNodeLogsHandler(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status %d want %d body=%s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			var body struct {
				Logs []struct {
					Level   string `json:"level"`
					Message string `json:"message"`
				} `json:"logs"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if len(body.Logs) != tc.wantLogs {
				t.Fatalf("logs %d want %d", len(body.Logs), tc.wantLogs)
			}
		})
	}
}
