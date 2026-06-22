package service

import (
	"context"
	"testing"
	"time"

	"github.com/klouddb/klouddbshield/model"
	"github.com/klouddb/klouddbshield/pkg/postgresdb"
	"github.com/klouddb/klouddbshield/pkg/reportstore"
)

func TestDecodeSSLResults(t *testing.T) {
	tests := []struct {
		name    string
		report  map[string]interface{}
		wantNil bool
		wantLen int
	}{
		{
			name:    "missing key",
			report:  map[string]interface{}{},
			wantNil: true,
		},
		{
			name: "valid ssl report",
			report: map[string]interface{}{
				"SSL Report": map[string]interface{}{
					"cells": []*model.SSLScanResultCell{
						{Title: "SSL Enabled Check", Status: "Pass"},
						{Title: "Self-Signed Certificate Check", Status: "Critical", Message: "self-signed"},
					},
					"ssl_params": map[string]string{"ssl_ciphers": "HIGH"},
					"hba_lines":  []string{"host all all 0.0.0.0/0 scram-sha-256"},
				},
			},
			wantLen: 2,
		},
		{
			name: "empty cells and params",
			report: map[string]interface{}{
				"SSL Report": map[string]interface{}{"cells": []interface{}{}},
			},
			wantNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := decodeSSLResults(tc.report)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("decodeSSLResults() = %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("decodeSSLResults() = nil, want result")
			}
			if len(got.Cells) != tc.wantLen {
				t.Fatalf("cells len = %d, want %d", len(got.Cells), tc.wantLen)
			}
		})
	}
}

func TestNormalizeSSLStatus(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Pass", "pass"},
		{"WARNING", "warning"},
		{"warn", "warning"},
		{"Critical", "critical"},
		{"Fail", "fail"},
		{"", "fail"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			if got := normalizeSSLStatus(tc.in); got != tc.want {
				t.Fatalf("normalizeSSLStatus(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSslRows(t *testing.T) {
	tests := []struct {
		name     string
		cis      []model.Result
		hba      []model.HBAScannerResult
		ssl      *model.SSLScanResult
		wantKeys []string
		minRows  int
	}{
		{
			name: "cis ssl controls allowlist",
			cis: []model.Result{
				{Control: "6.7", Title: "FIPS", Status: "Fail"},
				{Control: "6.8", Title: "SSL enabled", Status: "Pass"},
				{Control: "6.9", Title: "TLS versions", Status: "Pass"},
				{Control: "6.10", Title: "Weak ciphers", Status: "Fail"},
				{Control: "3.1", Title: "log_connections", Status: "Pass"},
			},
			wantKeys: []string{"6.7", "6.8", "6.9", "6.10"},
			minRows:  4,
		},
		{
			name: "ssl audit cells merged",
			cis:  []model.Result{{Control: "6.8", Title: "SSL", Status: "Pass"}},
			ssl: &model.SSLScanResult{
				Cells: []*model.SSLScanResultCell{
					{Title: "Self-Signed Certificate Check", Status: "Critical"},
				},
			},
			minRows: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rows := sslRows(tc.cis, tc.hba, tc.ssl)
			if len(rows) < tc.minRows {
				t.Fatalf("sslRows() len = %d, want >= %d", len(rows), tc.minRows)
			}
			for _, key := range tc.wantKeys {
				found := false
				for _, r := range rows {
					if len(r.Cells) > 0 && r.Cells[0] == key {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("missing CIS control row %q in %+v", key, rows)
				}
			}
		})
	}
}

func TestSslScannerFromSQLite(t *testing.T) {
	db := OpenTestSQLiteDB(t)
	defer db.Close()

	fileData := map[string]interface{}{
		"SSL Report": &model.SSLScanResult{
			Cells: []*model.SSLScanResultCell{
				{Title: "SSL Enabled Check", Status: "Pass"},
				{Title: "Self-Signed Certificate Check", Status: "Critical", Message: "self-signed"},
				{Title: "SSL Certificate Expiry Check", Status: "Warning", Message: "expires in 30 days"},
				{Title: "SSL HBA Check", Status: "Fail", Message: "host without hostssl"},
			},
			SSLParams: map[string]string{"ssl_ciphers": "HIGH", "ssl_cert_file": "/etc/ssl/cert.pem"},
			HBALines:  []string{"host all all 0.0.0.0/0 scram-sha-256"},
		},
	}
	pg := &postgresdb.Postgres{Host: "ssl-host", Port: "5432", DBName: "db"}
	PersistTestScanResult(t, db, fileData, reportstore.RunMeta{
		Trigger:     "test",
		RunnerName:  "ssl_check",
		FeaturesRun: []string{"ssl_check"},
		Postgres:    pg,
		StartedAt:   time.Now().UTC(),
		FinishedAt:  time.Now().UTC(),
		RunStatus:   "success",
	}, "test-node", pg.Host)

	tests := []struct {
		name         string
		wantAvail    bool
		wantPass     int
		wantFail     int
		wantWarning  int
		wantCritical int
		wantCells    int
	}{
		{
			name:         "ssl scanner counts",
			wantAvail:    true,
			wantPass:     1,
			wantFail:     1,
			wantWarning:  1,
			wantCritical: 1,
			wantCells:    4,
		},
	}

	svc := NewSQLiteService(db)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := svc.SslScanner(context.Background(), "ssl-host:5432")
			if err != nil {
				t.Fatal(err)
			}
			if resp.Available != tc.wantAvail {
				t.Fatalf("Available = %v, want %v", resp.Available, tc.wantAvail)
			}
			if resp.Pass != tc.wantPass {
				t.Fatalf("Pass = %d, want %d", resp.Pass, tc.wantPass)
			}
			if resp.Fail != tc.wantFail {
				t.Fatalf("Fail = %d, want %d", resp.Fail, tc.wantFail)
			}
			if resp.Warning != tc.wantWarning {
				t.Fatalf("Warning = %d, want %d", resp.Warning, tc.wantWarning)
			}
			if resp.Critical != tc.wantCritical {
				t.Fatalf("Critical = %d, want %d", resp.Critical, tc.wantCritical)
			}
			if len(resp.Cells) != tc.wantCells {
				t.Fatalf("Cells = %d, want %d", len(resp.Cells), tc.wantCells)
			}
			if resp.SSLParams["ssl_ciphers"] != "HIGH" {
				t.Fatalf("SSLParams = %+v", resp.SSLParams)
			}
			if len(resp.HBALines) != 1 {
				t.Fatalf("HBALines = %+v", resp.HBALines)
			}
		})
	}
}
