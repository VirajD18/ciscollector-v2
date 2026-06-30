package service

import (
	"strings"
	"testing"

	"github.com/VirajD18/ciscollector-v2/model"
	"github.com/VirajD18/ciscollector-v2/pkg/reportstore"
)

func TestResolveCriticalFromCIS(t *testing.T) {
	tests := []struct {
		name   string
		id     int
		result model.Result
		want   string
	}{
		{
			name: "scram pass from manual description",
			id:   1,
			result: model.Result{
				Status: "Manual",
				Title:  "Ensure Password Complexity is configured",
				ManualCheckData: model.ManualCheckTableDescriptionAndList{
					Description: "password_encryption: scram-sha-256",
				},
			},
			want: "Pass",
		},
		{
			name: "scram fail from manual description",
			id:   1,
			result: model.Result{
				Status: "Manual",
				ManualCheckData: model.ManualCheckTableDescriptionAndList{
					Description: "password_encryption: md5",
				},
			},
			want: "Fail",
		},
		{
			name: "md5 roles empty table pass",
			id:   2,
			result: model.Result{
				Status: "Manual",
				ManualCheckData: model.ManualCheckTableDescriptionAndList{
					Table: &model.SimpleTable{
						Columns: []string{"rolname"},
						Rows:    nil,
					},
				},
			},
			want: "Pass",
		},
		{
			name: "listen_addresses star fail",
			id:   9,
			result: model.Result{
				Status: "Manual",
				ManualCheckData: model.ManualCheckTableDescriptionAndList{
					List: []string{"*"},
				},
			},
			want: "Fail",
		},
		{
			name: "log_connections on pass",
			id:   11,
			result: model.Result{
				Status: "Manual",
				ManualCheckData: model.ManualCheckTableDescriptionAndList{
					Table: &model.SimpleTable{
						Columns: []string{"name", "setting"},
						Rows:    [][]interface{}{{"log_connections", "on"}},
					},
				},
			},
			want: "Pass",
		},
		{
			name: "cis fail preserved",
			id:   11,
			result: model.Result{
				Status:     "Fail",
				FailReason: "log_connections is not enabled",
			},
			want: "Fail",
		},
		{
			name: "public db rows fail",
			id:   24,
			result: model.Result{
				Status: "Manual",
				ManualCheckData: model.ManualCheckTableDescriptionAndList{
					Table: &model.SimpleTable{
						Columns: []string{"datname"},
						Rows:    [][]interface{}{{"customers"}},
					},
				},
			},
			want: "Fail",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveCriticalFromCIS(tc.id, tc.result)
			if !strings.EqualFold(got, tc.want) {
				t.Fatalf("resolveCriticalFromCIS() = %q want %q", got, tc.want)
			}
		})
	}
}

func TestCriticalChecksManualCISResolved(t *testing.T) {
	run := &reportstore.RunRow{
		Report: map[string]interface{}{
			"Postgres Report": map[string]interface{}{
				"result": []map[string]interface{}{
					{
						"Status":    "Manual",
						"Control":   "5.3",
						"Title":     "Ensure Password Complexity is configured",
						"Procedure": "SHOW password_encryption",
						"ManualCheckData": map[string]interface{}{
							"Description": "password_encryption: scram-sha-256",
							"Table": map[string]interface{}{
								"Columns": []string{"usename", "passwd"},
								"Rows":    nil,
							},
						},
					},
					{
						"Status":    "Manual",
						"Control":   "5.2",
						"Title":     "Ensure PostgreSQL is Bound to an IP Address",
						"Procedure": "SHOW listen_addresses",
						"ManualCheckData": map[string]interface{}{
							"List": []string{"localhost"},
						},
					},
				},
			},
		},
	}
	checks := CriticalChecksForRun(run)
	statusByID := map[int]string{}
	for _, c := range checks {
		statusByID[c.ID] = c.Status
	}
	if statusByID[1] != "Pass" {
		t.Fatalf("check 1 SCRAM: got %q want Pass", statusByID[1])
	}
	if statusByID[9] != "Pass" {
		t.Fatalf("check 9 listen_addresses: got %q want Pass", statusByID[9])
	}
}
