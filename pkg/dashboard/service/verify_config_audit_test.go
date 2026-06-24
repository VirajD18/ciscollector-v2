package service

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestConfigAuditRowsIncludePassAndFail(t *testing.T) {
	dbPath := filepath.Join(os.Getenv("USERPROFILE"), ".klouddb", "klouddbshield.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Skip("no live db")
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	resp, err := NewSQLiteService(db).FleetCategories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var cat *FleetCategory
	for i := range resp.Categories {
		if resp.Categories[i].ID == "config-audit" {
			cat = &resp.Categories[i]
			break
		}
	}
	if cat == nil {
		t.Fatal("no config-audit category")
	}
	var passN, failN int
	for _, row := range cat.Rows {
		if len(row) < 3 {
			continue
		}
		switch row[2] {
		case "Pass":
			passN++
		case "Fail":
			failN++
		default:
			t.Logf("other status %q: %v", row[2], row)
		}
	}
	if passN == 0 || failN == 0 {
		t.Fatalf("expected both Pass and Fail rows from SQLite, got pass=%d fail=%d total=%d", passN, failN, len(cat.Rows))
	}
	if !strings.Contains(cat.Count, "hosts") {
		t.Fatalf("count should still be host-based: %s", cat.Count)
	}
}
