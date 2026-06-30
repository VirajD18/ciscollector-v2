package service

import (
	"context"
	"testing"
	"time"

	"github.com/VirajD18/ciscollector-v2/model"
	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
	"github.com/VirajD18/ciscollector-v2/pkg/reportstore"
)

func TestStrategicFromResultArray(t *testing.T) {
	db := OpenTestSQLiteDB(t)
	defer db.Close()

	results := make([]interface{}, 0, 26)
	for i := 0; i < 16; i++ {
		results = append(results, map[string]interface{}{"Status": "Pass", "Control": "3.1.x", "Critical": false})
	}
	for i := 0; i < 10; i++ {
		results = append(results, map[string]interface{}{"Status": "Fail", "Control": "3.1.y", "Critical": false})
	}
	fileData := map[string]interface{}{
		"Postgres Report": map[string]interface{}{
			"result":  results,
			"version": "18",
		},
		"HBA Report": []model.HBAScannerResult{
			{Control: 1, Title: "Trust auth", Status: "Fail"},
			{Control: 8, Title: "hostssl", Status: "Fail"},
			{Control: 9, Title: "Open CIDR", Status: "Fail"},
		},
	}
	reportstore.RunsTable = "scan_results"
	_, err := reportstore.PersistScanResult(context.Background(), db, fileData, reportstore.ScanResultMeta{
		RunMeta: reportstore.RunMeta{
			Trigger: "manual", StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(),
			Postgres: &postgresdb.Postgres{Host: "localhost", Port: "5432"},
		},
		NodeID: "test-node", Hostname: "localhost",
	})
	if err != nil {
		t.Fatal(err)
	}

	svc := NewSQLiteService(db)
	resp, err := svc.Strategic(context.Background(), "30d")
	if err != nil {
		t.Fatal(err)
	}
	r := resp.Ranges["30d"]
	if r.CIS < 60 || r.CIS > 63 {
		t.Fatalf("cis=%d want ~61", r.CIS)
	}
	if r.Health < 60 || r.Health > 63 {
		t.Fatalf("health=%d want ~61", r.Health)
	}
	if r.Critical != 4 {
		t.Fatalf("critical=%d want 4 critical check failures (trust, open cidr, ssl, hostssl)", r.Critical)
	}
	if r.Servers != 1 {
		t.Fatalf("servers=%d", r.Servers)
	}
}
