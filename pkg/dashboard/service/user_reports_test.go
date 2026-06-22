package service

import (
	"context"
	"testing"
	"time"

	"github.com/klouddb/klouddbshield/pkg/postgresdb"
	"github.com/klouddb/klouddbshield/pkg/reportstore"
)

func TestUserReportsGroupsByInstance(t *testing.T) {
	db := OpenTestSQLiteDB(t)
	defer db.Close()
	now := time.Now().UTC()
	dbs := []string{"hej", "hej1", "hej3"}
	for _, name := range dbs {
		r := inactiveUsersLogParserReport()
		r["Log Parser Summary"] = []interface{}{
			map[string]interface{}{
				"Command": "inactive_users",
				"Result":  "2 inactive users found in database\n",
				"Value": []interface{}{
					[]interface{}{"postgres"},
					[]interface{}{"romin"},
					[]interface{}{"postgres", "romin"},
				},
			},
		}
		PersistTestScanResult(t, db, r, reportstore.RunMeta{
			Trigger:    "test",
			RunnerName: "inactive_users",
			Postgres:   &postgresdb.Postgres{Host: "localhost", Port: "5432", DBName: name},
			StartedAt:  now,
			FinishedAt: now,
			RunStatus:  "success",
		}, "test-node", "localhost")
	}

	svc := NewSQLiteService(db)
	inactive, err := svc.InactiveUsersReport(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if inactive.HostCount != 1 {
		t.Fatalf("hostCount=%d want 1", inactive.HostCount)
	}
	if inactive.UserCount != 2 {
		t.Fatalf("userCount=%d want 2", inactive.UserCount)
	}
	if len(inactive.Rows) != 2 {
		t.Fatalf("rows=%d want 2 (grouped by instance)", len(inactive.Rows))
	}
	for _, row := range inactive.Rows {
		if row.Instance != "localhost:5432" {
			t.Fatalf("instance=%q", row.Instance)
		}
		if row.DatabasesLabel != "3 (hej, hej1, hej3)" {
			t.Fatalf("databases_label=%q", row.DatabasesLabel)
		}
	}
}

func TestUserReportsFleet(t *testing.T) {
	tests := []struct {
		name              string
		report            map[string]interface{}
		wantInactiveHosts int
		wantInactiveUsers int
		wantCommonHosts   int
		wantCommonUsers   int
		wantInactiveMsg   bool
		wantCommonMsg     bool
	}{
		{
			name:              "no data",
			report:            map[string]interface{}{},
			wantInactiveMsg:   true,
			wantCommonMsg:     true,
		},
		{
			name:              "inactive users from log parser",
			report:            inactiveUsersLogParserReport(),
			wantInactiveHosts: 1,
			wantInactiveUsers: 2,
			wantCommonHosts:   1,
			wantCommonUsers:   2,
		},
		{
			name:            "common users from users report",
			report:          samplePostgresUsersReport(),
			wantCommonHosts: 1,
			wantCommonUsers: 2,
			wantCommonMsg:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := OpenTestSQLiteDB(t)
			defer db.Close()

			now := time.Now().UTC()
			PersistTestScanResult(t, db, tc.report, reportstore.RunMeta{
				Trigger:    "test",
				RunnerName: "postgres_cis",
				Postgres:   &postgresdb.Postgres{Host: "localhost", Port: "5432", DBName: "postgres"},
				StartedAt:  now,
				FinishedAt: now,
				RunStatus:  "success",
			}, "test-node", "localhost")

			svc := NewSQLiteService(db)
			inactive, err := svc.InactiveUsersReport(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if inactive.HostCount != tc.wantInactiveHosts {
				t.Fatalf("inactive hostCount %d want %d rows=%v", inactive.HostCount, tc.wantInactiveHosts, inactive.Rows)
			}
			if inactive.UserCount != tc.wantInactiveUsers {
				t.Fatalf("inactive userCount %d want %d", inactive.UserCount, tc.wantInactiveUsers)
			}
			if tc.wantInactiveMsg && inactive.Message == "" {
				t.Fatal("inactive message want non-empty")
			}

			common, err := svc.CommonUsersReport(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if common.HostCount != tc.wantCommonHosts {
				t.Fatalf("common hostCount %d want %d rows=%v", common.HostCount, tc.wantCommonHosts, common.Rows)
			}
			if common.UserCount != tc.wantCommonUsers {
				t.Fatalf("common userCount %d want %d", common.UserCount, tc.wantCommonUsers)
			}
			if tc.wantCommonMsg && common.Message == "" {
				t.Fatal("common message want non-empty")
			}
		})
	}
}

func TestInactiveUsersReportSeparateCronPush(t *testing.T) {
	db := OpenTestSQLiteDB(t)
	defer db.Close()

	now := time.Now().UTC()
	target := &postgresdb.Postgres{Host: "localhost", Port: "5432", DBName: "hej"}

	// Newer CIS-only push (no log parser) — mimics */2 cron overwriting log-parser tick.
	PersistTestScanResult(t, db, samplePostgresUsersReport(), reportstore.RunMeta{
		Trigger:    "cron",
		RunnerName: "postgres_cis",
		Postgres:   target,
		StartedAt:  now,
		FinishedAt: now,
		RunStatus:  "success",
	}, "test-node", "localhost")

	older := now.Add(-5 * time.Minute)
	PersistTestScanResult(t, db, inactiveUsersLogParserReport(), reportstore.RunMeta{
		Trigger:    "cron",
		RunnerName: "inactive_users",
		Postgres:   target,
		StartedAt:  older,
		FinishedAt: older,
		RunStatus:  "success",
	}, "test-node", "localhost")

	svc := NewSQLiteService(db)
	inactive, err := svc.InactiveUsersReport(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if inactive.UserCount != 2 {
		t.Fatalf("inactive userCount %d want 2 rows=%v", inactive.UserCount, inactive.Rows)
	}
}

func inactiveUsersLogParserReport() map[string]interface{} {
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
	return report
}
