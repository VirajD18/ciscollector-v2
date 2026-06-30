package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/VirajD18/ciscollector-v2/model"
	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
	"github.com/VirajD18/ciscollector-v2/pkg/reportstore"
)

func TestCriticalChecksTable(t *testing.T) {
	tests := []struct {
		name       string
		report     map[string]interface{}
		pii        map[string]interface{}
		port       string
		wantFails  []int
		wantPass   []int
		wantManual []int
	}{
		{
			name: "superuser count and common username",
			report: map[string]interface{}{
				"Users Report": []map[string]interface{}{
					{
						"Title": "Roles with Superuser attribute",
						"Data": map[string]interface{}{
							"Table": map[string]interface{}{
								"Columns": []string{"rolname"},
								"Rows":    [][]interface{}{{"postgres"}, {"romin"}, {"dba_admin"}, {"ops_lead"}, {"backup_svc"}},
							},
						},
					},
					{
						"Title": "List of db users",
						"Data": map[string]interface{}{
							"Table": map[string]interface{}{
								"Columns": []string{"rolname", "rolcanlogin"},
								"Rows":    [][]interface{}{{"postgres", true}, {"admin", true}},
							},
						},
					},
				},
			},
			port:      "5432",
			wantFails: []int{7, 18, 25},
		},
		{
			name: "hba trust and open network",
			report: map[string]interface{}{
				"HBA Report": []model.HBAScannerResult{
					{Control: 1, Title: "Trust auth", Status: "Fail", Description: "trust method found"},
					{Control: 9, Title: "Open CIDR", Status: "Fail", Description: "0.0.0.0/0 present"},
				},
			},
			wantFails: []int{8, 10},
		},
		{
			name: "cis logging checks",
			report: map[string]interface{}{
				"Postgres Report": map[string]interface{}{
					"result": []map[string]interface{}{
						{"Status": "Fail", "Control": "3.1.7", "Title": "Ensure log_connections is enabled", "FailReason": "log_connections is off"},
						{"Status": "Pass", "Control": "3.1.8", "Title": "Ensure log_disconnections is enabled"},
					},
				},
			},
			wantFails: []int{11},
			wantPass:  []int{12},
		},
		{
			name: "users report scram password_encryption",
			report: map[string]interface{}{
				"Users Report": []map[string]interface{}{
					{
						"Title": "SCRAM-SHA-256 password_encryption",
						"Data": map[string]interface{}{
							"Table": map[string]interface{}{
								"Columns": []string{"password_encryption", "pass"},
								"Rows":    [][]interface{}{{"scram-sha-256", true}},
							},
						},
					},
				},
			},
			wantPass: []int{1},
		},
		{
			name: "users report guc and public db checks",
			report: map[string]interface{}{
				"Users Report": []map[string]interface{}{
					{
						"Title": "log_connections enabled (Top 25 check #11)",
						"Data": map[string]interface{}{
							"Table": map[string]interface{}{
								"Columns": []string{"setting", "pass"},
								"Rows":    [][]interface{}{{"off", false}},
							},
						},
					},
					{
						"Title": "Databases open to PUBLIC connect",
						"Data": map[string]interface{}{
							"Table": map[string]interface{}{
								"Columns": []string{"datname"},
								"Rows":    [][]interface{}{{"appdb"}},
							},
						},
					},
					{
						"Title": "Roles with non-SCRAM password hashes",
						"Data": map[string]interface{}{
							"Table": map[string]interface{}{
								"Columns": []string{"non_scram_count", "pass"},
								"Rows":    [][]interface{}{{"2", false}},
							},
						},
					},
					{
						"Title": "SECURITY DEFINER functions (Top 25 check #6)",
						"Data": map[string]interface{}{
							"Table": map[string]interface{}{
								"Columns": []string{"secdef_count", "pass"},
								"Rows":    [][]interface{}{{"0", true}},
							},
						},
					},
				},
			},
			wantFails: []int{2, 11, 24},
			wantPass:  []int{6},
		},
		{
			name: "pii and password leak",
			report: map[string]interface{}{
				"Log Parser Summary": []interface{}{
					map[string]interface{}{"title": "Leaked password in pg_log", "command": "password_scan"},
				},
			},
			pii: map[string]interface{}{
				"high_confidence": map[string]interface{}{
					"columns": []interface{}{"database", "table", "column"},
					"rows":    []interface{}{[]interface{}{"customers", "contacts", "email"}},
				},
			},
			wantFails: []int{3, 4},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			run := &reportstore.RunRow{
				TargetPort: tc.port,
				Report:     tc.report,
				PiiReport:  tc.pii,
			}
			checks := CriticalChecksForRun(run)
			if len(checks) != len(criticalCheckTitles) {
				t.Fatalf("checks len %d want %d", len(checks), len(criticalCheckTitles))
			}
			statusByID := map[int]string{}
			for _, c := range checks {
				statusByID[c.ID] = c.Status
			}
			for _, id := range tc.wantFails {
				if !strings.EqualFold(statusByID[id], "Fail") {
					t.Fatalf("check %d status %q want Fail", id, statusByID[id])
				}
			}
			for _, id := range tc.wantPass {
				if !strings.EqualFold(statusByID[id], "Pass") {
					t.Fatalf("check %d status %q want Pass", id, statusByID[id])
				}
			}
			for _, id := range tc.wantManual {
				if !strings.EqualFold(statusByID[id], "Manual") {
					t.Fatalf("check %d status %q want Manual", id, statusByID[id])
				}
			}
		})
	}
}

func TestCriticalChecksFleetFromSQLite(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	now := time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC)
	report := samplePostgresUsersReport()
	report["HBA Report"] = []model.HBAScannerResult{
		{Control: 8, Title: "hostssl", Status: "Fail", Description: "host without ssl"},
	}
	PersistTestScanResult(t, db, report, reportstore.RunMeta{
		Trigger:    "test",
		RunnerName: "postgres_cis",
		Postgres:   &postgresdb.Postgres{Host: "localhost", Port: "5432"},
		StartedAt:  now,
		FinishedAt: now,
		RunStatus:  "success",
	}, "test-node", "localhost")

	resp, err := NewSQLiteService(db).CriticalChecksFleet(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Checks) != 25 {
		t.Fatalf("definitions %d want 25", len(resp.Checks))
	}
	if len(resp.HostRows) != 1 {
		t.Fatalf("host_rows %d want 1", len(resp.HostRows))
	}
	if len(resp.Rows) == 0 {
		t.Fatal("expected failing rows")
	}
	if len(resp.CheckOptions) != 25 {
		t.Fatalf("check_options %d want 25", len(resp.CheckOptions))
	}
	foundHostssl := false
	for _, row := range resp.Rows {
		if row.CheckID == 19 {
			foundHostssl = true
		}
	}
	if !foundHostssl {
		t.Fatal("expected hostssl failure row")
	}
}

func TestCriticalChecksPIIFallbackFromOlderRun(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ctx := context.Background()
	now := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	pg := &postgresdb.Postgres{Host: "localhost", Port: "5432", DBName: "hej"}
	emptyReport := map[string]interface{}{"Postgres Report": map[string]interface{}{}}

	PersistTestScanResult(t, db, emptyReport, reportstore.RunMeta{
		Trigger: "cron", RunnerName: "test", Postgres: pg,
		StartedAt: now.Add(-2 * time.Minute), FinishedAt: now.Add(-2 * time.Minute),
		RunStatus: "success",
	}, "test-node", pg.Host)
	if err := reportstore.PersistPIIReport(ctx, db, pg, map[string]interface{}{
		"high_confidence": map[string]interface{}{
			"columns": []interface{}{"database", "table", "column"},
			"rows":    []interface{}{[]interface{}{"hej", "contacts", "email"}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	PersistTestScanResult(t, db, emptyReport, reportstore.RunMeta{
		Trigger: "cron", RunnerName: "test", Postgres: pg,
		StartedAt: now, FinishedAt: now,
		RunStatus: "success",
	}, "test-node", pg.Host)

	latest, err := reportstore.GetLatestRun(ctx, db, reportstore.TargetID(pg))
	if err != nil || latest == nil {
		t.Fatal(err)
	}
	if len(latest.PiiReport) != 0 {
		t.Fatal("latest CIS run should not embed pii_report_json")
	}

	svc := NewSQLiteService(db)
	checks := svc.criticalChecksForRun(ctx, latest)
	statusByID := map[int]string{}
	for _, c := range checks {
		statusByID[c.ID] = c.Status
	}
	if !strings.EqualFold(statusByID[3], "Fail") {
		t.Fatalf("check 3 with PII fallback want Fail got %q", statusByID[3])
	}

	legacy := CriticalChecksForRun(latest)
	for _, c := range legacy {
		if c.ID == 3 && !strings.EqualFold(c.Status, "Pass") {
			t.Fatalf("CriticalChecksForRun without DB should Pass check 3, got %q", c.Status)
		}
	}
}
