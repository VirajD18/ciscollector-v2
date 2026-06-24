package reportstore

import (
	"context"
	"database/sql"
)

func configureSQLite(ctx context.Context, db *sql.DB) {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		_, _ = db.ExecContext(ctx, p)
	}
}
