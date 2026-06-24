package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestHbaScannerPerHostLiveDB(t *testing.T) {
	dbPath := filepath.Join(os.Getenv("USERPROFILE"), ".klouddb", "klouddbshield.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Skip("no live db")
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	svc := NewSQLiteService(db)
	ctx := context.Background()

	seen := map[string]string{}
	for _, host := range []string{"localhost:5432", "localhost:5434", "localhost:5435"} {
		r, err := svc.HbaScanner(ctx, host)
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Checks) == 0 {
			t.Logf("%s: no hba checks", host)
			continue
		}
		sig := fmtSignature(r)
		seen[host] = sig
		t.Logf("%s: checks=%d pass=%d fail=%d sig=%s", host, len(r.Checks), r.Pass, r.Fail, sig)
	}
	if len(seen) >= 2 {
		allSame := true
		var first string
		for _, sig := range seen {
			if first == "" {
				first = sig
			} else if sig != first {
				allSame = false
				break
			}
		}
		if allSame {
			t.Log("all hosts share identical HBA pass/fail signature in SQLite (same pg_hba rules)")
		}
	}
}

func fmtSignature(r *HbaScannerResponse) string {
	return fmt.Sprintf("%d/%d:%d", r.Pass, r.Fail, len(r.Checks))
}
