package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/VirajD18/ciscollector-v2/htmlreport"
	"github.com/VirajD18/ciscollector-v2/pkg/config"
	cons "github.com/VirajD18/ciscollector-v2/pkg/const"
	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
)

func TestGetProcessorsForCron_allIncludesPii(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "postgres.log")
	if err := os.WriteFile(logPath, []byte("2026-06-18 12:00:00 LOG: test\n"), 0o600); err != nil {
		t.Fatalf("write log file: %v", err)
	}

	pg := &postgresdb.Postgres{
		Host:     "localhost",
		Port:     "5432",
		DBName:   "hej",
		User:     "postgres",
		Password: "postgres",
	}
	shield := &config.Config{
		Postgres:   pg,
		PiiScanner: config.PiiScannerInput{RunOption: "datascan", Schema: "public"},
	}
	cmd := &config.Command{
		Name:     cons.RootCMD_All,
		Postgres: []*postgresdb.Postgres{pg},
		LogParser: &config.LogParserCronInput{
			Prefix:      "%t ",
			LogFile:     logPath,
			HbaConfFile: filepath.Join(t.TempDir(), "pg_hba.conf"),
		},
	}

	tests := []struct {
		name      string
		shield    *config.Config
		wantMin   int
		wantTypes []string
	}{
		{
			name:    "all bundle includes pii runner",
			shield:  shield,
			wantMin: 6,
			wantTypes: []string{
				"*main.postgresRunner",
				"*main.hbaRunner",
				"*main.sslAuditor",
				"*main.pwnedUserRunner",
				"*main.logParserRunner",
				"*main.piiDbScanner",
			},
		},
		{
			name:    "all without shield config skips pii",
			shield:  nil,
			wantMin: 5,
			wantTypes: []string{
				"*main.postgresRunner",
				"*main.hbaRunner",
				"*main.sslAuditor",
				"*main.pwnedUserRunner",
				"*main.logParserRunner",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileData := map[string]interface{}{}
			htmlHelperMap := htmlreport.NewHtmlReportHelperMap()
			runners, err := getProcessorsForCron("0 12 * * *", cmd, htmlHelperMap, fileData, tt.shield)
			if err != nil {
				t.Fatalf("getProcessorsForCron: %v", err)
			}
			if len(runners) < tt.wantMin {
				t.Fatalf("got %d runners, want at least %d", len(runners), tt.wantMin)
			}
			gotTypes := make([]string, len(runners))
			for i, r := range runners {
				gotTypes[i] = formatRunnerType(r)
			}
			for _, want := range tt.wantTypes {
				if !containsString(gotTypes, want) {
					t.Fatalf("missing runner %q in %v", want, gotTypes)
				}
			}
		})
	}
}

func formatRunnerType(r Runner) string {
	switch r.(type) {
	case *postgresRunner:
		return "*main.postgresRunner"
	case *hbaRunner:
		return "*main.hbaRunner"
	case *sslAuditor:
		return "*main.sslAuditor"
	case *pwnedUserRunner:
		return "*main.pwnedUserRunner"
	case *logParserRunner:
		return "*main.logParserRunner"
	case *piiDbScanner:
		return "*main.piiDbScanner"
	default:
		return "unknown"
	}
}

func containsString(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}
