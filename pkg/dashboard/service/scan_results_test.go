package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/VirajD18/ciscollector-v2/pkg/reportstore"
)

func TestLatestRunsByTargetDedupesScanResults(t *testing.T) {
	db := OpenTestSQLiteDB(t)
	defer db.Close()
	ctx := context.Background()

	fileData := map[string]interface{}{
		"Postgres Report": map[string]interface{}{
			"version": "16",
			"result":  []map[string]interface{}{{"Control": "1.1", "Status": "Pass"}},
		},
	}
	pg := reportstore.PostgresFromTarget("localhost", "5432", "hej")
	for i := 0; i < 3; i++ {
		PersistTestScanResult(t, db, fileData, reportstore.RunMeta{
			Trigger:    "cron",
			RunnerName: "ciscollector",
			Postgres:   pg,
			StartedAt:  time.Now().UTC().Add(time.Duration(i) * time.Minute),
			FinishedAt: time.Now().UTC(),
			RunStatus:  "success",
		}, "node-1", "host-a")
	}
	svc := NewSQLiteService(db)
	runs, err := svc.latestRunsByTarget(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("latestRunsByTarget count=%d want 1", len(runs))
	}
}

func TestLatestRunsByTargetPrefersSubstantiveReportOverPIIStub(t *testing.T) {
	db := OpenTestSQLiteDB(t)
	defer db.Close()
	ctx := context.Background()
	pg := reportstore.PostgresFromTarget("localhost", "5432", "hej")
	started := time.Now().UTC()

	PersistTestScanResult(t, db, map[string]interface{}{}, reportstore.RunMeta{
		Trigger: "manual", RunnerName: "pii_scanner", Postgres: pg,
		StartedAt: started, FinishedAt: started, RunStatus: "success",
	}, "node-1", "host-a")

	fileData := map[string]interface{}{
		"Postgres Report": map[string]interface{}{
			"version": "16",
			"result":  []map[string]interface{}{{"Control": "1.1", "Status": "Pass"}},
		},
	}
	PersistTestScanResult(t, db, fileData, reportstore.RunMeta{
		Trigger: "manual", RunnerName: "ciscollector", Postgres: pg,
		StartedAt: started, FinishedAt: started.Add(time.Second), RunStatus: "success",
	}, "node-1", "host-a")

	runs, err := NewSQLiteService(db).latestRunsByTarget(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("count=%d want 1", len(runs))
	}
	if _, ok := runs[0].Report["Postgres Report"]; !ok {
		t.Fatalf("expected Postgres Report in latest run, got keys=%v", runs[0].Report)
	}
}

func TestLatestRunsByTargetIncludesAllTargetsWithManyRuns(t *testing.T) {
	tests := []struct {
		name        string
		hostCount   int
		runsPerHost int
		wantTargets int
	}{
		{name: "fifty hosts with noisy history", hostCount: 50, runsPerHost: 12, wantTargets: 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := OpenTestSQLiteDB(t)
			defer db.Close()
			ctx := context.Background()

			base := time.Now().UTC()
			for i := 0; i < tt.hostCount; i++ {
				host := fmt.Sprintf("pg-%02d", i)
				pg := reportstore.PostgresFromTarget(host, "5432", "shielddb")
				for j := 0; j < tt.runsPerHost; j++ {
					PersistTestScanResult(t, db, map[string]interface{}{
						"Postgres Report": map[string]interface{}{
							"result": []interface{}{
								map[string]interface{}{"Status": "Pass", "Control": "1.1"},
							},
						},
					}, reportstore.RunMeta{
						Trigger:    "cron",
						RunnerName: "ciscollector",
						Postgres:   pg,
						StartedAt:  base.Add(time.Duration(i*tt.runsPerHost+j) * time.Second),
						FinishedAt: base.Add(time.Duration(i*tt.runsPerHost+j) * time.Second),
						RunStatus:  "success",
					}, fmt.Sprintf("node-%d", i), host)
				}
			}

			runs, err := NewSQLiteService(db).latestRunsByTarget(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(runs) != tt.wantTargets {
				t.Fatalf("latestRunsByTarget count=%d want %d", len(runs), tt.wantTargets)
			}
		})
	}
}
