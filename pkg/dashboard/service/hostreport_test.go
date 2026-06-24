package service

import (
	"context"
	"testing"
	"time"

	"github.com/klouddb/klouddbshield/model"
	"github.com/klouddb/klouddbshield/pkg/postgresdb"
	"github.com/klouddb/klouddbshield/pkg/reportstore"
)

func TestHostReportModules(t *testing.T) {
	db := OpenTestSQLiteDB(t)
	defer db.Close()

	fileData := map[string]interface{}{
		"Postgres Report": map[string]interface{}{
			"version": "16",
			"result": []model.Result{
				{Control: "3.1.20", Title: "log_connections enabled", Status: "Pass"},
				{Control: "6.8", Title: "SSL configured", Status: "Fail", Critical: true},
				{Control: "shared_preload_libraries", Title: "shared_preload_libraries", Status: "Fail"},
			},
		},
		"HBA Report": []model.HBAScannerResult{
			{Control: 2, Title: "HBA Check 2", Status: "Fail", Description: "all database"},
		},
		"SSL Report": &model.SSLScanResult{
			Cells: []*model.SSLScanResultCell{
				{Title: "SSL Enabled Check", Status: "Pass"},
			},
			SSLParams: map[string]string{"ssl_ciphers": "HIGH"},
		},
		"Log Parser Summary": []interface{}{
			map[string]interface{}{
				"Command": "inactive_users", "Parse Status": "All lines parsed successfully",
				"Result": "3 inactive users found in database\n", "title": "Inactive Users in DB",
			},
		},
	}
	pg := &postgresdb.Postgres{Host: "10.0.0.1", Port: "5432", DBName: "testdb"}
	PersistTestScanResult(t, db, fileData, reportstore.RunMeta{
		Trigger:    "test",
		RunnerName: "test",
		StartedAt:  time.Now().UTC(),
		FinishedAt: time.Now().UTC(),
		RunStatus:  "success",
		Postgres:   pg,
	}, "test-node", pg.Host)

	svc := NewSQLiteService(db)
	resp, err := svc.HostReport(context.Background(), "10.0.0.1:5432")
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("nil response")
	}
	if !resp.Modules.CisAudit.Available {
		t.Fatal("cis should be available")
	}
	if !resp.Modules.PgHba.Available {
		t.Fatal("hba should be available")
	}
	if !resp.Modules.SslTls.Available {
		t.Fatal("ssl_tls should be available")
	}
	if resp.SSLScanResult == nil {
		t.Fatal("ssl_scan_result should be set")
	}
	if !resp.Modules.LogParser.Available {
		t.Fatal("log parser should be available")
	}
	if resp.Modules.PiiResults.Available {
		t.Fatal("pii should be empty")
	}
	if len(resp.CriticalChecks) != 25 {
		t.Fatalf("critical_checks %d want 25", len(resp.CriticalChecks))
	}
}

func TestHostReportResolvesByTargetDB(t *testing.T) {
	db := OpenTestSQLiteDB(t)
	defer db.Close()

	fileData := samplePostgresUsersReport()
	pg := &postgresdb.Postgres{Host: "localhost", Port: "5436", DBName: "analytics_guest"}
	PersistTestScanResult(t, db, fileData, reportstore.RunMeta{
		Trigger:    "test",
		RunnerName: "postgres_cis",
		Postgres:   pg,
		StartedAt:  time.Now().UTC(),
		FinishedAt: time.Now().UTC(),
		RunStatus:  "success",
	}, "test-node", pg.Host)

	resp, err := NewSQLiteService(db).HostReport(context.Background(), "analytics_guest")
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected host report when server id matches target_db")
	}
	if resp.Host.Name != "localhost:5436" {
		t.Fatalf("host name %q want localhost:5436", resp.Host.Name)
	}
}
