package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/VirajD18/ciscollector-v2/model"
	"github.com/VirajD18/ciscollector-v2/pkg/reportstore"
)

func TestRenderRunHTML(t *testing.T) {
	tests := []struct {
		name     string
		run      *reportstore.RunRow
		wantSub  []string
		wantFail bool
	}{
		{
			name:     "nil run",
			run:      nil,
			wantSub:  []string{"No run data"},
			wantFail: false,
		},
		{
			name: "full template report",
			run: &reportstore.RunRow{
				ID:        "run-1",
				StartedAt: time.Date(2026, 6, 8, 11, 0, 0, 0, time.UTC),
				Report: map[string]interface{}{
					"Postgres Report": map[string]interface{}{
						"version": "16",
						"result": []model.Result{
							{Control: "3.1.2", Title: "log destinations", Status: "Pass"},
							{Control: "3.1.6", Title: "log file permissions", Status: "Fail", FailReason: "log_file_mode"},
						},
					},
					"HBA Report": []model.HBAScannerResult{
						{Control: 1, Title: "Trust auth", Status: "Fail"},
					},
				},
			},
			wantSub: []string{
				"KloudDBShield",
				"Critical Violations",
				"Postgres Security Report",
				"HBA Scanner Report",
				"Expand All",
			},
			wantFail: false,
		},
		{
			name: "critical violations in simple fallback",
			run: &reportstore.RunRow{
				ID:        "run-cv",
				StartedAt: time.Now().UTC(),
				Report: map[string]interface{}{
					"Postgres Report": map[string]interface{}{
						"result": []model.Result{
							{Control: "5.1", Title: "trust", Status: "Fail"},
						},
					},
				},
			},
			wantSub: []string{
				"Critical Violations",
				"Violation Details",
			},
			wantFail: false,
		},
		{
			name: "empty report falls back to simple html",
			run: &reportstore.RunRow{
				ID:        "run-2",
				StartedAt: time.Now().UTC(),
				Report:    map[string]interface{}{},
			},
			wantSub:  []string{"KloudDB Shield"},
			wantFail: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := RenderRunHTML(context.Background(), nil, tc.run)
			if got == "" {
				t.Fatal("empty html output")
			}
			for _, sub := range tc.wantSub {
				if !strings.Contains(got, sub) {
					t.Fatalf("output missing %q\n%s", sub, got[:min(400, len(got))])
				}
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
