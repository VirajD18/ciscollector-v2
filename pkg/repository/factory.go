package repository

import (
	"context"
	"fmt"

	"github.com/klouddb/klouddbshield/pkg/repository/postgres"
	"github.com/klouddb/klouddbshield/pkg/repository/sqlite"
)

// Open creates a repository for the configured storage driver.
func Open(ctx context.Context, cfg Config) (Repository, error) {
	cfg = ResolveFromEnv(cfg)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	switch cfg.EffectiveDriver() {
	case DriverPostgres:
		return postgres.Open(ctx, cfg.PostgresURL)
	case DriverSQLite:
		return sqlite.Open(ctx, cfg.SQLitePath)
	default:
		return nil, fmt.Errorf("unsupported storage driver %q", cfg.Driver)
	}
}

// OpenSQLiteAtPath opens a SQLite repository at an existing path (tests).
func OpenSQLiteAtPath(ctx context.Context, path string) (Repository, error) {
	return sqlite.Open(ctx, path)
}
