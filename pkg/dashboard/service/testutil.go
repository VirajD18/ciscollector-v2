package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
	"github.com/VirajD18/ciscollector-v2/pkg/reportstore"
	sqliterepo "github.com/VirajD18/ciscollector-v2/pkg/repository/sqlite"
	_ "modernc.org/sqlite"
)

// NewSQLiteService wraps a SQLite *sql.DB as a dashboard Service (tests and legacy callers).
func NewSQLiteService(db *sql.DB) *Service {
	reportstore.RunsTable = "scan_results"
	repo := sqliterepo.FromDB(db)
	_ = repo.EnsureSchema(context.Background())
	return New(repo)
}

// NewSQLiteServiceWithConfig wraps SQLite with an optional kshieldconfig path.
func NewSQLiteServiceWithConfig(db *sql.DB, configPath string) *Service {
	reportstore.RunsTable = "scan_results"
	repo := sqliterepo.FromDB(db)
	_ = repo.EnsureSchema(context.Background())
	return NewWithConfig(repo, configPath)
}

// OpenTestSQLiteDB opens a SQLite DB with scan_results schema (tests).
func OpenTestSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", "file:"+path+"?mode=rwc&_journal_mode=WAL")
	if err != nil {
		t.Fatal(err)
	}
	reportstore.RunsTable = "scan_results"
	if err := reportstore.EnsureScanResultsSchema(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return db
}

// PersistTestReport inserts one scan_results row for dashboard tests.
func PersistTestReport(t *testing.T, db *sql.DB, report map[string]interface{}, host string) {
	PersistTestReportWithTime(t, db, report, host, time.Now().UTC())
}

func PersistTestReportWithTime(t *testing.T, db *sql.DB, report map[string]interface{}, host string, ts time.Time) {
	PersistTestScanResult(t, db, report, reportstore.RunMeta{
		Trigger:    "test",
		RunnerName: "postgres_cis",
		Postgres:   &postgresdb.Postgres{Host: host, Port: "5432"},
		StartedAt:  ts,
		FinishedAt: ts,
		RunStatus:  "success",
	}, "test-node", host)
}

func PersistTestScanResult(t *testing.T, db *sql.DB, report map[string]interface{}, meta reportstore.RunMeta, nodeID, hostname string) {
	t.Helper()
	reportstore.RunsTable = "scan_results"
	_, err := reportstore.PersistScanResult(context.Background(), db, report, reportstore.ScanResultMeta{
		RunMeta:  meta,
		NodeID:   nodeID,
		Hostname: hostname,
	})
	if err != nil {
		t.Fatal(err)
	}
}
