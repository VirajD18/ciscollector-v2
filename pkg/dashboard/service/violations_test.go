package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/klouddb/klouddbshield/model"
)

func TestViolationsRowsHaveDetectedAt(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	now := time.Date(2026, 5, 26, 6, 30, 0, 0, time.UTC)
	PersistTestReportWithTime(t, db, samplePostgresUsersReport(), "localhost", now)

	v, err := NewSQLiteService(db).Violations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Rows) < 3 {
		t.Fatalf("rows %d want >=3", len(v.Rows))
	}
	if v.Rows[0].Detected != "2026-05-26 06:30" {
		t.Fatalf("detected %q", v.Rows[0].Detected)
	}
	if v.Rows[0].Type != "Critical Config" {
		t.Fatalf("type %q", v.Rows[0].Type)
	}

	for _, want := range []string{"Critical Config", "PII Exposure", "SSL Violation", "Password Leak", "Unauthorized Superuser"} {
		if !sliceContains(v.TypeOptions, want) {
			t.Fatalf("type_options missing %q: %v", want, v.TypeOptions)
		}
	}
	if len(v.SeverityOptions) == 0 || !sliceContains(v.SeverityOptions, "HIGH") {
		t.Fatalf("severity_options %v", v.SeverityOptions)
	}
}

func TestViolationDetailsSkipsCmdNoise(t *testing.T) {
	r := model.Result{
		Control: "1.2",
		Title:   "Ensure PostgreSQL is enabled and running",
		Status:  "Fail",
		FailReason: "\n\t\tcmd: systemctl is-enabled postgresql@17-main.service\n" +
			"\t\tcmderr: exit status 1\n\t\touterr: WSL ERROR CreateProcessEntryCommon",
	}
	d := violationDetailsFromCIS(r)
	if strings.Contains(strings.ToLower(d), "cmd:") || strings.Contains(d, "WSL") {
		t.Fatalf("details should not include shell noise: %q", d)
	}
	if d == "" {
		t.Fatal("expected title-based details")
	}
}

func TestViolationDetailsUsesShortFailReason(t *testing.T) {
	r := model.Result{
		Control:    "3.1.3",
		Title:      "Ensure 'logging_collector' is enabled",
		Status:     "Fail",
		FailReason: "logging_collector is not enabled",
	}
	d := violationDetailsFromCIS(r)
	if d != "logging_collector is not enabled" {
		t.Fatalf("got %q", d)
	}
}

func sliceContains(ss []string, x string) bool {
	for _, s := range ss {
		if s == x {
			return true
		}
	}
	return false
}
