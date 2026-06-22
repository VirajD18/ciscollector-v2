package logparser

import (
	"testing"

	cons "github.com/klouddb/klouddbshield/pkg/const"
)

func TestBuildReadinessReport(t *testing.T) {
	tests := []struct {
		name                   string
		in                     ReadinessInput
		wantConn               string
		wantCISLMA             string
		wantPrefixCIS          string
		wantCanRunMenu         string
		wantRunnable           int
		wantLogParserReadiness string
	}{
		{
			name: "full CIS pass all menus",
			in: ReadinessInput{
				LogConnectionsOn: true,
				LogLinePrefix:    "%m %p %l %d %u %a %h",
				Commands:         defaultLogParserCommands,
			},
			wantConn:               "on",
			wantCISLMA:             "pass",
			wantPrefixCIS:          "pass",
			wantCanRunMenu:         "6 · 7 · 8 · 10",
			wantRunnable:           4,
			wantLogParserReadiness: "PASS",
		},
		{
			name: "log_connections off blocks inactive_users",
			in: ReadinessInput{
				LogConnectionsOn: false,
				LogLinePrefix:    "%m %p %l %d %u %a %h",
				Commands:         []string{cons.LogParserCMD_InactiveUser, cons.LogParserCMD_UniqueIPs},
			},
			wantConn:               "off",
			wantCISLMA:             "partial",
			wantPrefixCIS:          "pass",
			wantCanRunMenu:         "6 · 7",
			wantRunnable:           2,
			wantLogParserReadiness: "PASS",
		},
		{
			name: "legacy host no GUCs",
			in: ReadinessInput{
				LogConnectionsOn: false,
				LogLinePrefix:    "minimal",
				Commands:         defaultLogParserCommands,
			},
			wantConn:               "off",
			wantCISLMA:             "fail",
			wantPrefixCIS:          "fail",
			wantCanRunMenu:         "10",
			wantRunnable:           1,
			wantLogParserReadiness: "PASS",
		},
		{
			name: "unique_ip only with log_connections",
			in: ReadinessInput{
				LogConnectionsOn: true,
				LogLinePrefix:    "no %u/%a/%h",
				Commands:         []string{cons.LogParserCMD_UniqueIPs, cons.LogParserCMD_InactiveUser},
			},
			wantConn:               "on",
			wantCISLMA:             "partial",
			wantCanRunMenu:         "6 · 7",
			wantRunnable:           2,
			wantLogParserReadiness: "PASS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildReadinessReport(tt.in)
			if got["log_connections"] != tt.wantConn {
				t.Fatalf("log_connections = %v, want %v", got["log_connections"], tt.wantConn)
			}
			if tt.wantCISLMA != "" && got["cis_lma"] != tt.wantCISLMA {
				t.Fatalf("cis_lma = %v, want %v", got["cis_lma"], tt.wantCISLMA)
			}
			if tt.wantPrefixCIS != "" && got["log_line_prefix_cis"] != tt.wantPrefixCIS {
				t.Fatalf("log_line_prefix_cis = %v, want %v", got["log_line_prefix_cis"], tt.wantPrefixCIS)
			}
			if got["can_run_menus"] != tt.wantCanRunMenu {
				t.Fatalf("can_run_menus = %v, want %v", got["can_run_menus"], tt.wantCanRunMenu)
			}
			runnable, _ := got["runnable_commands"].([]string)
			if len(runnable) != tt.wantRunnable {
				t.Fatalf("runnable_commands len = %d, want %d", len(runnable), tt.wantRunnable)
			}
			if tt.wantLogParserReadiness != "" && got["logparser_readiness"] != tt.wantLogParserReadiness {
				t.Fatalf("logparser_readiness = %v, want %v", got["logparser_readiness"], tt.wantLogParserReadiness)
			}
		})
	}
}

func TestEvaluateLogParserReadiness(t *testing.T) {
	tests := []struct {
		name          string
		runnableCount int
		wantReadiness string
	}{
		{name: "no runnable commands", runnableCount: 0, wantReadiness: "FAIL"},
		{name: "one runnable command", runnableCount: 1, wantReadiness: "PASS"},
		{name: "multiple runnable commands", runnableCount: 4, wantReadiness: "PASS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateLogParserReadiness(tt.runnableCount)
			if got != tt.wantReadiness {
				t.Fatalf("evaluateLogParserReadiness(%d) = %q, want %q", tt.runnableCount, got, tt.wantReadiness)
			}
		})
	}
}
