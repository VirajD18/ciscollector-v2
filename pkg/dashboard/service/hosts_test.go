package service

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/klouddb/klouddbshield/pkg/postgresdb"
	"github.com/klouddb/klouddbshield/pkg/reportstore"
	sqliterepo "github.com/klouddb/klouddbshield/pkg/repository/sqlite"
)

func TestParseHostKey(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		wantInstance string
		wantDB       string
		wantHostKey  string
	}{
		{
			name:         "full host key",
			key:          "localhost:5432/hej",
			wantInstance: "localhost:5432",
			wantDB:       "hej",
			wantHostKey:  "localhost:5432/hej",
		},
		{
			name:         "instance only",
			key:          "localhost:5432",
			wantInstance: "localhost:5432",
			wantHostKey:  "localhost:5432",
		},
		{
			name:         "target id",
			key:          "postgres:localhost:5432:hej1",
			wantInstance: "localhost:5432",
			wantDB:       "hej1",
			wantHostKey:  "localhost:5432/hej1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseHostKey(tc.key)
			if got.Instance != tc.wantInstance || got.Database != tc.wantDB || got.HostKey != tc.wantHostKey {
				t.Fatalf("ParseHostKey(%q) = %+v want instance=%q db=%q hostKey=%q",
					tc.key, got, tc.wantInstance, tc.wantDB, tc.wantHostKey)
			}
		})
	}
}

func TestHostsGroupsByInstance(t *testing.T) {
	tests := []struct {
		name          string
		targets       []postgresdb.Postgres
		wantInstances int
		wantDBCount   int
	}{
		{
			name: "two databases same instance",
			targets: []postgresdb.Postgres{
				{Host: "localhost", Port: "5432", DBName: "hej"},
				{Host: "localhost", Port: "5432", DBName: "hej1"},
			},
			wantInstances: 1,
			wantDBCount:   2,
		},
		{
			name: "different instances stay separate",
			targets: []postgresdb.Postgres{
				{Host: "localhost", Port: "5432", DBName: "hej"},
				{Host: "host.docker.internal", Port: "5450", DBName: "testdb"},
			},
			wantInstances: 2,
			wantDBCount:   1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prev := reportstore.RunsTable
			reportstore.RunsTable = "scan_results"
			t.Cleanup(func() { reportstore.RunsTable = prev })

			dir := t.TempDir()
			db, err := reportstore.Open(filepath.Join(dir, "test.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := reportstore.EnsureScanResultsSchema(context.Background(), db); err != nil {
				t.Fatal(err)
			}

			started := time.Now().UTC().Add(-time.Minute)
			for _, pg := range tc.targets {
				meta := reportstore.ScanResultMeta{
					RunMeta: reportstore.RunMeta{
						Trigger: "cron", RunnerName: "test", Postgres: &pg,
						StartedAt: started, FinishedAt: started, RunStatus: "success",
					},
					NodeID: "n1", Hostname: "host1",
				}
				_, err = reportstore.PersistScanResult(context.Background(), db, map[string]interface{}{
					"Postgres Report": []interface{}{
						map[string]interface{}{"Status": "Fail", "Control": 1, "Title": "check"},
					},
				}, meta)
				if err != nil {
					t.Fatal(err)
				}
			}

			repo, err := sqliterepo.Open(context.Background(), filepath.Join(dir, "test.db"))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = repo.Close() })

			svc := New(repo)
			resp, err := svc.Hosts(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(resp.Instances) != tc.wantInstances {
				t.Fatalf("instances=%d want %d", len(resp.Instances), tc.wantInstances)
			}
			if tc.wantInstances == 1 {
				inst := resp.Instances[0]
				if inst.DatabaseCount != tc.wantDBCount {
					t.Fatalf("database_count=%d want %d", inst.DatabaseCount, tc.wantDBCount)
				}
				if !strings.Contains(inst.DatabasesLabel, "hej") {
					t.Fatalf("databases_label=%q", inst.DatabasesLabel)
				}
			}
		})
	}
}

func TestInstancePostureLabel(t *testing.T) {
	tests := []struct {
		name    string
		failing int
		total   int
		want    string
	}{
		{name: "all passing", failing: 0, total: 2, want: "Passing"},
		{name: "partial failing", failing: 1, total: 2, want: "1/2 Failing"},
		{name: "all failing", failing: 2, total: 2, want: "2/2 Failing"},
		{name: "no databases", failing: 0, total: 0, want: "-"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := instancePostureLabel(tc.failing, tc.total); got != tc.want {
				t.Fatalf("instancePostureLabel(%d,%d)=%q want %q", tc.failing, tc.total, got, tc.want)
			}
		})
	}
}
