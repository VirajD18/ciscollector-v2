package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
	"github.com/VirajD18/ciscollector-v2/pkg/reportstore"
)

func TestFleetCategoriesFromSampleReport(t *testing.T) {
	report := samplePostgresUsersReport()
	db := OpenTestSQLiteDB(t)
	defer db.Close()

	now := time.Now().UTC()
	PersistTestScanResult(t, db, report, reportstore.RunMeta{
		Trigger:    "test",
		RunnerName: "postgres_cis",
		Postgres:   &postgresdb.Postgres{Host: "localhost", Port: "5432", DBName: "postgres"},
		StartedAt:  now,
		FinishedAt: now,
		RunStatus:  "success",
	}, "test-node", "localhost")

	svc := NewSQLiteService(db)
	resp, err := svc.FleetCategories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Categories) != 12 {
		t.Fatalf("categories %d want 12", len(resp.Categories))
	}
	byID := map[string]FleetCategory{}
	for _, c := range resp.Categories {
		byID[c.ID] = c
	}
	if byID["cis-benchmarks"].Count != "1 hosts" {
		t.Fatalf("cis: %s", byID["cis-benchmarks"].Count)
	}
	if byID["ssl-violations"].Count != "0 hosts" {
		t.Fatalf("ssl: %s", byID["ssl-violations"].Count)
	}
	if byID["elevated-privs"].Count != "1 hosts" {
		t.Fatalf("elevated: %s rows=%d", byID["elevated-privs"].Count, len(byID["elevated-privs"].Rows))
	}
	if byID["config-audit"].Count != "1 hosts" {
		t.Fatalf("config: %s", byID["config-audit"].Count)
	}
	cfg := byID["config-audit"].Rows
	if len(cfg) < 3 {
		t.Fatalf("config rows %d want pass+fail checks", len(cfg))
	}
	var hasPass, hasFail bool
	for _, row := range cfg {
		if len(row) < 4 {
			continue
		}
		resultCol := 3
		checkCol := 2
		if row[resultCol] == "Pass" {
			hasPass = true
		}
		if row[resultCol] == "Fail" {
			hasFail = true
		}
		if row[resultCol] == "Fail" && row[checkCol] == row[0] {
			t.Fatalf("check label should not be host only: %v", row)
		}
	}
	if !hasPass || !hasFail {
		t.Fatalf("config-audit needs Pass and Fail from report status, got rows=%v", cfg)
	}
	if byID["config-drift"].Count != "1 hosts" {
		t.Fatalf("drift: %s", byID["config-drift"].Count)
	}
	if byID["common-users"].Count != "2 users" {
		t.Fatalf("common-users: %s rows=%v", byID["common-users"].Count, byID["common-users"].Rows)
	}
	for _, row := range byID["common-users"].Rows {
		if len(row) < 4 || row[3] != "View detail" {
			t.Fatalf("common-users row: %v", row)
		}
		if row[1] != "localhost:5432" {
			t.Fatalf("common-users host column: %v", row)
		}
		if row[2] != "postgres" {
			t.Fatalf("common-users database column: %v", row)
		}
	}
	if byID["inactive-users"].Count != "0 hosts" {
		t.Fatalf("inactive-users without log parser: %s rows=%v", byID["inactive-users"].Count, byID["inactive-users"].Rows)
	}
	if byID["usage-of-defaults"].Count != "1 hosts" {
		t.Fatalf("usage-of-defaults on default port+postgres: %s rows=%v", byID["usage-of-defaults"].Count, byID["usage-of-defaults"].Rows)
	}
	if byID["hba-issues"].Count != "0 hosts" {
		t.Fatalf("hba-issues without hba report: %s", byID["hba-issues"].Count)
	}
	if byID["superuser-counts"].Count != "0 hosts" {
		t.Fatalf("superuser-counts within limit: %s rows=%v", byID["superuser-counts"].Count, byID["superuser-counts"].Rows)
	}
}

func TestFleetInactiveUsersFromLogParser(t *testing.T) {
	report := samplePostgresUsersReport()
	report["Log Parser Summary"] = []interface{}{
		map[string]interface{}{
			"Command": "inactive_users",
			"Result":  "2 inactive users found in database\n",
			"Value": []interface{}{
				[]interface{}{"postgres"},
				[]interface{}{"romin"},
				[]interface{}{"stale_a", "stale_b"},
			},
		},
	}
	db := OpenTestSQLiteDB(t)
	defer db.Close()
	now := time.Now().UTC()
	PersistTestScanResult(t, db, report, reportstore.RunMeta{
		Trigger:    "test",
		RunnerName: "postgres_cis",
		Postgres:   &postgresdb.Postgres{Host: "localhost", Port: "5432", DBName: "postgres"},
		StartedAt:  now,
		FinishedAt: now,
		RunStatus:  "success",
	}, "test-node", "localhost")

	resp, err := NewSQLiteService(db).FleetCategories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var inactive FleetCategory
	for _, c := range resp.Categories {
		if c.ID == "inactive-users" {
			inactive = c
			break
		}
	}
	if inactive.Count != "1 hosts" {
		t.Fatalf("inactive hosts: %s", inactive.Count)
	}
	if len(inactive.Rows) != 2 {
		t.Fatalf("inactive rows %v want stale_a and stale_b only", inactive.Rows)
	}
	users := map[string]bool{}
	for _, row := range inactive.Rows {
		users[row[0]] = true
		if strings.Contains(row[2], "password expiry") {
			t.Fatalf("password expiry must not count as inactive: %v", row)
		}
	}
	if !users["stale_a"] || !users["stale_b"] {
		t.Fatalf("missing inactive users: %v", inactive.Rows)
	}
}

func samplePostgresUsersReport() map[string]interface{} {
	return map[string]interface{}{
		"Postgres Report": map[string]interface{}{
			"result": []map[string]interface{}{
				{"Status": "Pass", "Control": "3.1.2", "Title": "log destinations", "Critical": false},
				{"Status": "Fail", "Control": "3.1.6", "Title": "log file permissions", "FailReason": "log_file_mode is not set correctly", "Critical": false},
				{"Status": "Fail", "Control": "3.2", "Title": "pgaudit", "FailReason": "pgaudit is not enabled", "Critical": false},
			},
		},
		"Users Report": []map[string]interface{}{
			{
				"Title": "Roles with Superuser attribute",
				"Data": map[string]interface{}{
					"Table": map[string]interface{}{
						"Columns": []string{"rolname"},
						"Rows":    [][]interface{}{{"postgres"}, {"romin"}},
					},
				},
			},
			{
				"Title": "List of db users",
				"Data": map[string]interface{}{
					"Table": map[string]interface{}{
						"Columns": []string{"rolname", "rolsuper", "rolinherit", "rolcreaterole", "rolcreatedb", "rolcanlogin"},
						"Rows": [][]interface{}{
							{"postgres", true, true, true, true, true},
							{"romin", true, true, false, false, true},
						},
					},
				},
			},
			{
				"Title": "Password expiry not set (Roles without password expiry)",
				"Data": map[string]interface{}{
					"Table": map[string]interface{}{
						"Columns": []string{"rolname"},
						"Rows":    [][]interface{}{{"postgres"}, {"romin"}},
					},
				},
			},
		},
	}
}
