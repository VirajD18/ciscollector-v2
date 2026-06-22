package reportstore

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const sqliteBusyTimeoutMs = 5000

func sqliteDSN(path string) string {
	return fmt.Sprintf("file:%s?mode=rwc&_busy_timeout=%d&_journal_mode=WAL&_txlock=immediate&_foreign_keys=on", path, sqliteBusyTimeoutMs)
}

// Open opens (or creates) the report database and applies migrations (read/write).
func Open(dbPath string) (*sql.DB, error) {
	return openDB(dbPath, false)
}

// EnsureMigrations applies pending schema updates (e.g. pii_report_json). Safe to call before OpenReadOnly.
func EnsureMigrations(dbPath string) error {
	db, err := openDB(dbPath, false)
	if err != nil {
		return err
	}
	return db.Close()
}

// OpenReadOnly opens the report DB for dashboard reads without blocking collector writes.
func OpenReadOnly(dbPath string) (*sql.DB, error) {
	return openDB(dbPath, true)
}

func openDB(dbPath string, readOnly bool) (*sql.DB, error) {
	dsn := sqliteDSN(dbPath)
	if readOnly {
		dsn = fmt.Sprintf("file:%s?mode=ro&_busy_timeout=%d", dbPath, sqliteBusyTimeoutMs)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if readOnly {
		db.SetMaxOpenConns(4)
	} else {
		db.SetMaxOpenConns(1)
	}
	if !readOnly {
		if err := migrate(context.Background(), db); err != nil {
			db.Close()
			return nil, err
		}
		configureSQLite(context.Background(), db)
	}
	return db, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	entries, err := fs.Glob(migrationsFS, "migrations/*.up.sql")
	if err != nil {
		return err
	}
	sort.Strings(entries)
	for _, path := range entries {
		raw, err := migrationsFS.ReadFile(path)
		if err != nil {
			return err
		}
		for _, stmt := range splitSQL(string(raw)) {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				if isIgnorableMigrationErr(err) {
					continue
				}
				return fmt.Errorf("migration %s: %w", path, err)
			}
		}
	}
	return nil
}

func isIgnorableMigrationErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column") ||
		strings.Contains(msg, "already exists")
}

func splitSQL(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
