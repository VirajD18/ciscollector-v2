package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/VirajD18/ciscollector-v2/model"
)

func TestNetworkConnectivityNoHBAOrSSL(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	report := map[string]interface{}{
		"Postgres Report": map[string]interface{}{
			"result": []map[string]interface{}{
				{"Status": "Pass", "Control": "3.1.2", "Title": "log destinations"},
			},
		},
	}
	persistReport(t, db, report, "localhost")

	r := strategicRange(t, db)
	if r.HBAScanned || len(r.HBA) != 0 {
		t.Fatalf("hba want empty got scanned=%v len=%d", r.HBAScanned, len(r.HBA))
	}
	if r.SSLScanned {
		t.Fatal("ssl should be empty without connection/ssl cis")
	}
}

func TestNetworkConnectivityPostgresLoggingReport(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	report := map[string]interface{}{
		"Postgres Report": map[string]interface{}{
			"result": []map[string]interface{}{
				{"Status": "Pass", "Control": "3.1.23", "Title": "Ensure 'log_hostname' is set correctly"},
				{"Status": "Fail", "Control": "3.1.20", "Title": "Ensure 'log_connections' is enabled"},
				{"Status": "Fail", "Control": "3.1.21", "Title": "Ensure 'log_disconnections' is enabled"},
				{"Status": "Fail", "Control": "3.1.24", "Title": "Ensure 'log_line_prefix' is set correctly"},
			},
			"version": "18",
		},
	}
	persistReport(t, db, report, "localhost")

	r := strategicRange(t, db)
	if !r.HBAScanned || len(r.HBA) != 1 {
		t.Fatalf("hba chart len=%d scanned=%v", len(r.HBA), r.HBAScanned)
	}
	if !r.SSLScanned || r.SSLEnforced != 25 {
		t.Fatalf("ssl enforced=%d want 25 (1 pass / 4 checks)", r.SSLEnforced)
	}
}

func TestNetworkConnectivityWithHBAAndSSL(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	report := samplePostgresUsersReport()
	report["HBA Report"] = []model.HBAScannerResult{
		{Control: 1, Title: "trust method", Status: "Fail"},
		{Control: 2, Title: "hostssl rule", Status: "Pass"},
	}
	pr := report["Postgres Report"].(map[string]interface{})
	pr["result"] = append(pr["result"].([]map[string]interface{}),
		map[string]interface{}{"Control": "6.8", "Title": "SSL enabled", "Status": "Fail"},
		map[string]interface{}{"Control": "6.9", "Title": "TLS certificate", "Status": "Pass"},
	)

	persistReport(t, db, report, "db-01")

	r := strategicRange(t, db)
	if !r.SSLScanned || r.SSLEnforced < 66 || r.SSLEnforced > 67 {
		t.Fatalf("ssl scanned=%v enforced=%d want ~67%% (2 pass / 3 ssl-related checks)", r.SSLScanned, r.SSLEnforced)
	}
	if !r.HBAScanned || len(r.HBA) != 1 {
		t.Fatalf("hba scanned=%v len=%d", r.HBAScanned, len(r.HBA))
	}
	if r.HBA[0].O != 50 || r.HBA[0].S != 50 {
		t.Fatalf("hba open/secure want 50/50 got O=%d S=%d I=%d", r.HBA[0].O, r.HBA[0].S, r.HBA[0].I)
	}
}

func openTestDB(t *testing.T) *sql.DB {
	return OpenTestSQLiteDB(t)
}

func persistReport(t *testing.T, db *sql.DB, report map[string]interface{}, host string) {
	PersistTestReport(t, db, report, host)
}

func strategicRange(t *testing.T, db *sql.DB) StrategicRange {
	t.Helper()
	resp, err := NewSQLiteService(db).Strategic(context.Background(), "30d")
	if err != nil {
		t.Fatal(err)
	}
	return resp.Ranges["30d"]
}
