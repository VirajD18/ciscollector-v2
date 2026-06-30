package service

import (
	"strings"
	"testing"

	"github.com/VirajD18/ciscollector-v2/pkg/reportstore"
)

func TestEvalFromUsersReportAllTop25SQLChecks(t *testing.T) {
	tests := []struct {
		name       string
		sections   []map[string]interface{}
		checkID    int
		wantStatus string
		wantSource string
	}{
		{
			name: "check1 scram pass",
			sections: []map[string]interface{}{usersTableSection("SCRAM-SHA-256 password_encryption",
				[]string{"password_encryption", "pass"}, [][]interface{}{{"scram-sha-256", true}})},
			checkID: 1, wantStatus: "Pass", wantSource: "Users Report",
		},
		{
			name: "check7 superuser fail",
			sections: []map[string]interface{}{usersTableSection("Superuser count (Top 25 check #7)",
				[]string{"superuser_count", "pass"}, [][]interface{}{{"5", false}})},
			checkID: 7, wantStatus: "Fail", wantSource: "Users Report",
		},
		{
			name: "check9 listen_addresses pass",
			sections: []map[string]interface{}{usersTableSection("listen_addresses not wildcard (Top 25 check #9)",
				[]string{"listen_addresses", "pass"}, [][]interface{}{{"localhost", true}})},
			checkID: 9, wantStatus: "Pass", wantSource: "Users Report",
		},
		{
			name: "check17 pgaudit.log fail",
			sections: []map[string]interface{}{usersTableSection("pgaudit.log configured (Top 25 check #17)",
				[]string{"pgaudit_log", "pass"}, [][]interface{}{{"ddl", false}})},
			checkID: 17, wantStatus: "Fail", wantSource: "Users Report",
		},
		{
			name: "check24 public db pass empty",
			sections: []map[string]interface{}{usersTableSection("Databases open to PUBLIC connect",
				[]string{"datname"}, nil)},
			checkID: 24, wantStatus: "Pass", wantSource: "Users Report",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			run := &reportstore.RunRow{
				Report: map[string]interface{}{"Users Report": tc.sections},
			}
			checks := CriticalChecksForRun(run)
			var got CriticalCheckResult
			for _, c := range checks {
				if c.ID == tc.checkID {
					got = c
					break
				}
			}
			if got.ID == 0 {
				t.Fatalf("check %d not found", tc.checkID)
			}
			if !strings.EqualFold(got.Status, tc.wantStatus) {
				t.Fatalf("status %q want %q details=%q", got.Status, tc.wantStatus, got.Details)
			}
			if got.Source != tc.wantSource {
				t.Fatalf("source %q want %q", got.Source, tc.wantSource)
			}
		})
	}
}

func usersTableSection(title string, cols []string, rows [][]interface{}) map[string]interface{} {
	rowIface := make([]interface{}, len(rows))
	for i, r := range rows {
		rowIface[i] = r
	}
	colIface := make([]interface{}, len(cols))
	for i, c := range cols {
		colIface[i] = c
	}
	return map[string]interface{}{
		"Title": title,
		"Data": map[string]interface{}{
			"Table": map[string]interface{}{
				"Columns": colIface,
				"Rows":    rowIface,
			},
		},
	}
}
