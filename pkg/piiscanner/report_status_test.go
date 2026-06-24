package piiscanner_test

import (
	"testing"

	"github.com/klouddb/klouddbshield/pkg/piiscanner"
)

func TestReportPayload(t *testing.T) {
	cnf := piiscanner.Config{Schema: "public"}

	tests := []struct {
		name       string
		out        *piiscanner.DatabasePIIScanOutput
		status     string
		message    string
		errDetail  string
		wantStatus string
		wantMsg    string
		wantErr    string
		wantEmpty  bool
	}{
		{
			name:       "no tables",
			status:     piiscanner.ReportStatusNoTables,
			message:    "No tables found in database matching the PII scan criteria.",
			wantStatus: piiscanner.ReportStatusNoTables,
			wantMsg:    "No tables found in database matching the PII scan criteria.",
			wantEmpty:  true,
		},
		{
			name:       "scan error",
			status:     piiscanner.ReportStatusError,
			message:    "PII scan failed.",
			errDetail:  "permission denied for schema secret",
			wantStatus: piiscanner.ReportStatusError,
			wantMsg:    "PII scan failed.",
			wantErr:    "permission denied for schema secret",
			wantEmpty:  true,
		},
		{
			name:       "success with no findings becomes no_data",
			out:        &piiscanner.DatabasePIIScanOutput{},
			status:     piiscanner.ReportStatusSuccess,
			wantStatus: piiscanner.ReportStatusNoData,
			wantMsg:    "No PII data found in database.",
			wantEmpty:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := piiscanner.ReportPayload(tt.out, cnf, tt.status, tt.message, tt.errDetail)
			if got["status"] != tt.wantStatus {
				t.Fatalf("status: got %v want %q", got["status"], tt.wantStatus)
			}
			if got["message"] != tt.wantMsg {
				t.Fatalf("message: got %v want %q", got["message"], tt.wantMsg)
			}
			if tt.wantErr != "" {
				if got["error"] != tt.wantErr {
					t.Fatalf("error: got %v want %q", got["error"], tt.wantErr)
				}
			}
			if tt.wantEmpty {
				hc, _ := got["high_confidence"].(map[string]interface{})
				if hc != nil {
					if rows, _ := hc["rows"].([]interface{}); len(rows) > 0 {
						t.Fatalf("expected empty high_confidence rows")
					}
				}
			}
		})
	}
}
