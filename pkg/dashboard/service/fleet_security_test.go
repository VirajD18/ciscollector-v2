package service

import (
	"context"
	"testing"
	"time"

	"github.com/klouddb/klouddbshield/model"
	"github.com/klouddb/klouddbshield/pkg/postgresdb"
	"github.com/klouddb/klouddbshield/pkg/reportstore"
)

func TestFleetSecurityCategories(t *testing.T) {
	tests := []struct {
		name            string
		report          map[string]interface{}
		port            string
		wantCategories  int
		wantHBAHosts    string
		wantDefaults    string
		wantSuperusers  string
		wantHBARows     int
		wantDefaultsRow int
		wantSuperRows   int
	}{
		{
			name:           "sample report flags default port and postgres role",
			report:         samplePostgresUsersReport(),
			port:           "5432",
			wantCategories: 12,
			wantHBAHosts:   "0 hosts",
			wantDefaults:   "1 hosts",
			wantSuperusers: "0 hosts",
		},
		{
			name: "hba failures from scanner report",
			report: map[string]interface{}{
				"HBA Report": []model.HBAScannerResult{
					{Control: 1, Title: "Trust auth", Status: "Fail"},
					{Control: 9, Title: "Open CIDR", Status: "Fail"},
				},
			},
			wantCategories: 12,
			wantHBAHosts:   "1 hosts",
			wantDefaults:   "0 hosts",
			wantSuperusers: "0 hosts",
			wantHBARows:    2,
		},
		{
			name: "superuser count exceeds limit",
			report: map[string]interface{}{
				"Users Report": []map[string]interface{}{
					{
						"Title": "Roles with Superuser attribute",
						"Data": map[string]interface{}{
							"Table": map[string]interface{}{
								"Columns": []string{"rolname"},
								"Rows": [][]interface{}{
									{"postgres"}, {"romin"}, {"dba_admin"}, {"ops_lead"}, {"backup_svc"},
								},
							},
						},
					},
					{
						"Title": "List of db users",
						"Data": map[string]interface{}{
							"Table": map[string]interface{}{
								"Columns": []string{"rolname", "rolcanlogin"},
								"Rows":    [][]interface{}{{"postgres", true}},
							},
						},
					},
				},
			},
			port:           "5432",
			wantCategories: 12,
			wantHBAHosts:   "0 hosts",
			wantDefaults:   "1 hosts",
			wantSuperusers: "1 hosts",
			wantSuperRows:  1,
		},
		{
			name: "top25 superuser count section fail",
			report: map[string]interface{}{
				"Users Report": []map[string]interface{}{
					usersTableSection("Superuser count (Top 25 check #7)",
						[]string{"superuser_count", "pass"}, [][]interface{}{{"5", false}}),
				},
			},
			wantCategories: 12,
			wantHBAHosts:   "0 hosts",
			wantDefaults:   "0 hosts",
			wantSuperusers: "1 hosts",
			wantSuperRows:  1,
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
				Postgres:   &postgresdb.Postgres{Host: "localhost", Port: tc.port, DBName: "postgres"},
				StartedAt:  now,
				FinishedAt: now,
				RunStatus:  "success",
			}, "test-node", "localhost")

			resp, err := NewSQLiteService(db).FleetCategories(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(resp.Categories) != tc.wantCategories {
				t.Fatalf("categories %d want %d", len(resp.Categories), tc.wantCategories)
			}
			byID := map[string]FleetCategory{}
			for _, c := range resp.Categories {
				byID[c.ID] = c
			}
			if byID["hba-issues"].Count != tc.wantHBAHosts {
				t.Fatalf("hba-issues: %s rows=%v", byID["hba-issues"].Count, byID["hba-issues"].Rows)
			}
			if byID["usage-of-defaults"].Count != tc.wantDefaults {
				t.Fatalf("usage-of-defaults: %s rows=%v", byID["usage-of-defaults"].Count, byID["usage-of-defaults"].Rows)
			}
			if byID["superuser-counts"].Count != tc.wantSuperusers {
				t.Fatalf("superuser-counts: %s rows=%v", byID["superuser-counts"].Count, byID["superuser-counts"].Rows)
			}
			if tc.wantHBARows > 0 && len(byID["hba-issues"].Rows) != tc.wantHBARows {
				t.Fatalf("hba rows %d want %d", len(byID["hba-issues"].Rows), tc.wantHBARows)
			}
			if tc.wantDefaultsRow > 0 && len(byID["usage-of-defaults"].Rows) != tc.wantDefaultsRow {
				t.Fatalf("defaults rows %d want %d", len(byID["usage-of-defaults"].Rows), tc.wantDefaultsRow)
			}
			if tc.wantSuperRows > 0 && len(byID["superuser-counts"].Rows) != tc.wantSuperRows {
				t.Fatalf("superuser rows %d want %d", len(byID["superuser-counts"].Rows), tc.wantSuperRows)
			}
		})
	}
}
