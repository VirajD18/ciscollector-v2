package service

import (
	"testing"

	"github.com/klouddb/klouddbshield/model"
)

func TestExtractGucNameFromTitle(t *testing.T) {
	r := model.Result{
		Title:  "Ensure 'log_connections' is enabled",
		Status: "Fail",
	}
	if got := extractGucName(r); got != "log_connections" {
		t.Fatalf("name=%q", got)
	}
}

func TestGucLiveValueEnabled(t *testing.T) {
	pass := model.Result{Title: "Ensure 'ssl' is enabled", Status: "Pass"}
	fail := model.Result{Title: "Ensure 'log_connections' is enabled", Status: "Fail"}
	if gucLiveValue(pass) != "on" {
		t.Fatalf("pass=%q", gucLiveValue(pass))
	}
	if gucLiveValue(fail) != "off" {
		t.Fatalf("fail=%q", gucLiveValue(fail))
	}
}

func TestLooksLikePostgresGucName(t *testing.T) {
	for _, bad := range []string{"1", "Postmaster", "SIGHUP", "/var/lib/postgresql/data/pg_hba.conf"} {
		if looksLikePostgresGucName(bad) {
			t.Fatalf("should reject %q", bad)
		}
	}
	for _, good := range []string{"log_connections", "ssl", "DateStyle", "max_connections"} {
		if !looksLikePostgresGucName(good) {
			t.Fatalf("should accept %q", good)
		}
	}
}

func TestGucValuesFromManualCheckRequiresPgSettingsColumns(t *testing.T) {
	r := model.Result{
		ManualCheckData: model.ManualCheckTableDescriptionAndList{
			Table: &model.SimpleTable{
				Columns: []string{"line_number", "file_name"},
				Rows:    [][]interface{}{{1, "/var/lib/postgresql/data/pg_hba.conf"}},
			},
		},
	}
	if len(gucValuesFromManualCheck(r)) != 0 {
		t.Fatal("hba table must not be parsed as GUCs")
	}
}

func TestGucDriftRowUsesSettingName(t *testing.T) {
	r := model.Result{
		Control:    "3.1.20",
		Title:      "Ensure 'log_connections' is enabled",
		Status:     "Fail",
		FailReason: "log_connections is not enabled",
	}
	name := extractGucName(r)
	if name != "log_connections" {
		t.Fatalf("name=%q", name)
	}
	if gucDriftLiveValue(r) == "" {
		t.Fatal("expected live value")
	}
	if gucExpectedFromCIS(r) != "on" {
		t.Fatalf("baseline=%q", gucExpectedFromCIS(r))
	}
}

