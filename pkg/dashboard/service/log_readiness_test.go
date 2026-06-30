package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/VirajD18/ciscollector-v2/pkg/dashboard/service"
	"github.com/VirajD18/ciscollector-v2/pkg/logparser"
	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
	"github.com/VirajD18/ciscollector-v2/pkg/reportstore"
)

func TestLogReadinessFleet(t *testing.T) {
	tests := []struct {
		name          string
		report        map[string]interface{}
		wantHost      string
		wantConn      string
		wantReadiness string
		wantRowCount  int
	}{
		{
			name: "explicit log readiness report",
			report: map[string]interface{}{
				logparser.LogReadinessReportKey: logparser.BuildReadinessReport(logparser.ReadinessInput{
					LogConnectionsOn: true,
					LogLinePrefix:    "%m %p %l %d %u %a %h",
				}),
			},
			wantHost:      "lr-host:5432/shielddb",
			wantConn:      "on",
			wantReadiness: "PASS",
			wantRowCount:  1,
		},
		{
			name: "fallback from guc settings",
			report: map[string]interface{}{
				"GUC Settings": map[string]interface{}{
					"settings": map[string]string{
						"log_connections": "off",
						"log_line_prefix": "minimal",
					},
				},
			},
			wantHost:      "lr-host:5432/shielddb",
			wantConn:      "off",
			wantReadiness: "PASS",
			wantRowCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := service.OpenTestSQLiteDB(t)
			defer db.Close()

			pg := &postgresdb.Postgres{Host: "lr-host", Port: "5432", DBName: "shielddb"}
			service.PersistTestScanResult(t, db, tt.report, reportstore.RunMeta{
				Trigger:    "test",
				RunnerName: "test",
				StartedAt:  time.Now().UTC(),
				FinishedAt: time.Now().UTC(),
				RunStatus:  "success",
				Postgres:   pg,
			}, "test-node", pg.Host)

			resp, err := service.NewSQLiteService(db).LogReadinessFleet(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(resp.Rows) != tt.wantRowCount {
				t.Fatalf("rows = %d, want %d", len(resp.Rows), tt.wantRowCount)
			}
			if tt.wantRowCount == 0 {
				return
			}
			row := resp.Rows[0]
			if row.Host != tt.wantHost {
				t.Fatalf("host = %q, want %q", row.Host, tt.wantHost)
			}
			if row.LogConnections != tt.wantConn {
				t.Fatalf("logConnections = %q, want %q", row.LogConnections, tt.wantConn)
			}
			if row.LogParserReadiness != tt.wantReadiness {
				t.Fatalf("logparserReadiness = %q, want %q", row.LogParserReadiness, tt.wantReadiness)
			}
		})
	}
}

func TestLogReadinessFleetIncludesAllHosts(t *testing.T) {
	tests := []struct {
		name      string
		hostCount int
	}{
		{name: "fifty host fleet", hostCount: 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := service.OpenTestSQLiteDB(t)
			defer db.Close()

			for i := 0; i < tt.hostCount; i++ {
				host := fmt.Sprintf("lr-%02d", i)
				pg := &postgresdb.Postgres{Host: host, Port: "5432", DBName: "shielddb"}
				service.PersistTestScanResult(t, db, map[string]interface{}{
					logparser.LogReadinessReportKey: logparser.BuildReadinessReport(logparser.ReadinessInput{
						LogConnectionsOn: true,
						LogLinePrefix:    "%m %p %l %d %u %a %h",
					}),
				}, reportstore.RunMeta{
					Trigger:    "test",
					RunnerName: "test",
					StartedAt:  time.Now().UTC().Add(time.Duration(i) * time.Second),
					FinishedAt: time.Now().UTC(),
					RunStatus:  "success",
					Postgres:   pg,
				}, fmt.Sprintf("node-%d", i), pg.Host)
			}

			resp, err := service.NewSQLiteService(db).LogReadinessFleet(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(resp.Rows) != tt.hostCount {
				t.Fatalf("rows = %d, want %d", len(resp.Rows), tt.hostCount)
			}
		})
	}
}
