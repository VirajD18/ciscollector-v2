package service

import (
	"context"
	"testing"
	"time"

	"github.com/klouddb/klouddbshield/pkg/postgresdb"
	"github.com/klouddb/klouddbshield/pkg/reportstore"
)

func TestComplianceFromCISReport(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	persistReport(t, db, samplePostgresUsersReport(), "localhost")

	r := strategicRange(t, db)
	if len(r.Audit) == 0 {
		t.Fatal("expected config audit rows")
	}
	if r.Audit[0][0] == "" || r.Audit[0][0] == "3.1.6" {
		t.Fatalf("audit should use fail reason label, got %q", r.Audit[0][0])
	}
	if len(r.Drift) == 0 {
		t.Fatal("expected drift series from scan history")
	}
	if r.PiiScanned {
		t.Fatal("pii should not be scanned without PII report")
	}
}

func TestConfigDriftFromLoggingCIS(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	report := map[string]interface{}{
		"Postgres Report": map[string]interface{}{
			"result": []map[string]interface{}{
				{"Status": "Pass", "Control": "3.1.2", "Title": "Ensure the log destinations are set correctly"},
				{"Status": "Fail", "Control": "3.1.20", "Title": "Ensure 'log_connections' is enabled"},
				{"Status": "Fail", "Control": "3.1.21", "Title": "Ensure 'log_disconnections' is enabled"},
				{"Status": "Fail", "Control": "3.1.24", "Title": "Ensure 'log_line_prefix' is set correctly"},
			},
		},
	}
	persistReport(t, db, report, "localhost")

	r := strategicRange(t, db)
	if len(r.Drift) != 1 || len(r.DriftLabels) != 1 {
		t.Fatalf("drift len=%d labels=%v", len(r.Drift), r.DriftLabels)
	}
	if r.Drift[0].B != 1 || r.Drift[0].D != 3 {
		t.Fatalf("drift bar want pass=1 fail=3 got b=%d d=%d", r.Drift[0].B, r.Drift[0].D)
	}
	if r.DriftLabels[0] != "localhost:5432" {
		t.Fatalf("label %q", r.DriftLabels[0])
	}
}

func TestConfigDriftFromGucBaseline(t *testing.T) {
	db := openTestGucDB(t)
	defer db.Close()

	ctx := context.Background()
	if err := reportstore.UpsertGucBaseline(ctx, db, "golden", map[string]string{
		"log_connections": "on",
		"log_statement":   "ddl",
	}); err != nil {
		t.Fatal(err)
	}
	if err := reportstore.UpsertServerGucSnapshot(ctx, db, "postgres:host-a:5432:db", "host-a", "node-1", map[string]string{
		"log_connections": "on",
		"log_statement":   "all",
	}); err != nil {
		t.Fatal(err)
	}
	if err := reportstore.UpsertServerGucSnapshot(ctx, db, "postgres:host-b:5432:db", "host-b", "node-2", map[string]string{
		"log_connections": "on",
		"log_statement":   "ddl",
	}); err != nil {
		t.Fatal(err)
	}

	drift, labels := buildGucDriftStrategicChart(ctx, NewSQLiteService(db))
	if len(drift) != 2 || len(labels) != 2 {
		t.Fatalf("drift=%v labels=%v", drift, labels)
	}
	if drift[0].B != 1 || drift[0].D != 1 {
		t.Fatalf("host-a want 1 matched 1 drifted got b=%d d=%d", drift[0].B, drift[0].D)
	}
	if drift[1].B != 2 || drift[1].D != 0 {
		t.Fatalf("host-b want 2 matched 0 drifted got b=%d d=%d", drift[1].B, drift[1].D)
	}
}

func TestPIIHeatmapFallsBackToPIIRun(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ctx := context.Background()
	pg := &postgresdb.Postgres{Host: "collector-1", Port: "5432", User: "postgres", DBName: "shielddb"}
	report := samplePostgresUsersReport()

	PersistTestScanResult(t, db, report, reportstore.RunMeta{
		Trigger: "test", RunnerName: "postgres_cis", Postgres: pg,
		StartedAt: time.Now().UTC().Add(-time.Minute), FinishedAt: time.Now().UTC().Add(-time.Minute),
		RunStatus: "success",
	}, "test-node", pg.Host)
	piiPayload := map[string]interface{}{
		"high_confidence": map[string]interface{}{
			"columns": []string{"table", "column", "label", "confidence"},
			"rows":    [][]interface{}{{"employees", "ssn", "SSN", "High"}},
		},
	}
	if err := reportstore.PersistPIIReport(ctx, db, pg, piiPayload); err != nil {
		t.Fatal(err)
	}

	// Newer CIS-only run without pii_report_json becomes "latest" per target.
	PersistTestScanResult(t, db, report, reportstore.RunMeta{
		Trigger:    "test",
		RunnerName: "postgres_cis",
		Postgres:   pg,
		StartedAt:  time.Now().UTC(),
		FinishedAt: time.Now().UTC(),
		RunStatus:  "success",
	}, "test-node", pg.Host)

	runs, err := NewSQLiteService(db).latestRunsByTarget(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || len(runs[0].PiiReport) != 0 {
		t.Fatalf("latest run should be CIS-only, pii len=%d", len(runs[0].PiiReport))
	}

	cols, grid, scanned := buildPIIHeatmap(ctx, NewSQLiteService(db).Repo, runs)
	if !scanned || len(cols) != 1 || grid[0][0] != 1 {
		t.Fatalf("heatmap scanned=%v cols=%v grid[0]=%v", scanned, cols, grid[0])
	}

	resp, err := NewSQLiteService(db).FleetCategories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var piiCat FleetCategory
	for _, c := range resp.Categories {
		if c.ID == "pii-violations" {
			piiCat = c
			break
		}
	}
	if len(piiCat.Rows) != 1 {
		t.Fatalf("pii-violations rows=%v", piiCat.Rows)
	}
}

func TestCompliancePIIHeatmap(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	report := samplePostgresUsersReport()
	persistReport(t, db, report, "localhost")
	pg := &postgresdb.Postgres{Host: "localhost", Port: "5432", User: "postgres", DBName: "postgres"}
	piiPayload := map[string]interface{}{
		"high_confidence": map[string]interface{}{
			"columns": []string{"table", "column", "label", "confidence", "detector", "matched"},
			"rows": [][]interface{}{
				{"employees", "ssn", "SSN", "High", "regex", "10/10"},
				{"employees", "phone_number", "Phone", "Moderate", "regex", "5/10"},
			},
		},
	}
	if err := reportstore.PersistPIIReport(context.Background(), db, pg, piiPayload); err != nil {
		t.Fatal(err)
	}

	r := strategicRange(t, db)
	if !r.PiiScanned {
		t.Fatal("expected pii scanned")
	}
	if len(r.HeatmapColumns) != 1 || r.HeatmapColumns[0] != "localhost:5432" {
		t.Fatalf("columns %v", r.HeatmapColumns)
	}
	if r.Heatmap[0][0] != 1 {
		t.Fatalf("high row want 1 got %v", r.Heatmap[0])
	}
	if r.Heatmap[1][0] != 1 {
		t.Fatalf("moderate row want 1 got %v", r.Heatmap[1])
	}
}

func TestPiiSeverityRow(t *testing.T) {
	tests := []struct {
		name string
		row  PiiScannerRow
		want int
	}{
		{name: "confidence high", row: PiiScannerRow{Confidence: "High"}, want: 0},
		{name: "confidence moderate", row: PiiScannerRow{Confidence: "Moderate"}, want: 1},
		{name: "confidence desirable", row: PiiScannerRow{Confidence: "Desirable"}, want: 2},
		{name: "confidence low", row: PiiScannerRow{Confidence: "Low"}, want: 3},
		{name: "legacy label high", row: PiiScannerRow{Label: "high"}, want: 0},
		{name: "default moderate", row: PiiScannerRow{Label: "Email"}, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := piiSeverityRow(tt.row); got != tt.want {
				t.Fatalf("piiSeverityRow() = %d want %d", got, tt.want)
			}
		})
	}
}
