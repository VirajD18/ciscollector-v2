package service

import (
	"testing"
)

func TestDecodeCISResultsWithManualCheckData(t *testing.T) {
	report := map[string]interface{}{
		"Postgres Report": map[string]interface{}{
			"result": []interface{}{
				map[string]interface{}{
					"Control": "3.1.20",
					"Title":   "Ensure 'log_connections' is enabled",
					"Status":  "Pass",
				},
				map[string]interface{}{
					"Control": "3.2.1",
					"Title":   "Roles",
					"Status":  "Fail",
					"ManualCheckData": map[string]interface{}{
						"Description": "review",
						"Table": map[string]interface{}{
							"Columns": []interface{}{"name", "setting"},
							"Rows":    []interface{}{[]interface{}{"log_connections", "on"}},
						},
					},
				},
			},
		},
	}
	cis := decodeCISResults(report)
	if len(cis) != 2 {
		t.Fatalf("len=%d", len(cis))
	}
	if cis[0].Title == "" {
		t.Fatal("first title empty")
	}
}
