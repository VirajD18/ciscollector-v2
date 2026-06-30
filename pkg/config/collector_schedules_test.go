package config

import (
	"strings"
	"testing"

	cons "github.com/VirajD18/ciscollector-v2/pkg/const"
	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
)

func TestCommandsFromPostgres_piiScanner(t *testing.T) {
	pg := &postgresdb.Postgres{
		Host:   "localhost",
		Port:   "5432",
		DBName: "hej",
	}
	lp := &LogParserCronInput{
		Prefix:  "%t ",
		LogFile: "/var/log/postgresql.log",
	}

	tests := []struct {
		name      string
		pg        *postgresdb.Postgres
		scanRaw   string
		lp        *LogParserCronInput
		wantNames []string
		wantErr   string
	}{
		{
			name:      "all expands to RootCMD_All bundle",
			scanRaw:   "all",
			pg:        pg,
			lp:        &LogParserCronInput{Prefix: "%t ", LogFile: "/var/log/postgresql.log", HbaConfFile: "/etc/postgresql/pg_hba.conf"},
			wantNames: []string{cons.RootCMD_All},
		},
		{
			name:      "all ignores trailing pii_scanner token",
			scanRaw:   "all,pii_scanner",
			pg:        pg,
			lp:        &LogParserCronInput{Prefix: "%t ", LogFile: "/var/log/postgresql.log", HbaConfFile: "/etc/postgresql/pg_hba.conf"},
			wantNames: []string{cons.RootCMD_All},
		},
		{
			name:      "pii_scanner accepted",
			scanRaw:   "pii_scanner",
			pg:        pg,
			wantNames: []string{cons.RootCMD_PiiScanner},
		},
		{
			name:      "pii_scanner with other commands",
			scanRaw:   "postgres_cis,hba_scanner,pii_scanner",
			pg:        pg,
			wantNames: []string{cons.RootCMD_PostgresCIS, cons.RootCMD_HBAScanner, cons.RootCMD_PiiScanner},
		},
		{
			name:      "inactive_users requires logfile",
			scanRaw:   "inactive_users",
			pg:        pg,
			lp:        &LogParserCronInput{},
			wantNames: []string{cons.LogParserCMD_InactiveUser},
			wantErr:   "logfile required",
		},
		{
			name:      "inactive_users with logparser config",
			scanRaw:   "inactive_users",
			pg:        pg,
			lp:        lp,
			wantNames: []string{cons.LogParserCMD_InactiveUser},
		},
		{
			name:    "unknown command rejected",
			scanRaw: "pii_scanner,unknown_cmd",
			pg:      pg,
			wantErr: `unsupported scan command "unknown_cmd"`,
		},
		{
			name:    "nil postgres rejected",
			scanRaw: "pii_scanner",
			pg:      nil,
			wantErr: "kshieldconfig",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useLP := tt.lp
			if useLP == nil && tt.scanRaw == "inactive_users" && tt.wantErr == "" {
				useLP = lp
			}
			cmds, err := CommandsFromPostgres(tt.pg, tt.scanRaw, useLP)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cmds) != len(tt.wantNames) {
				t.Fatalf("got %d commands, want %d", len(cmds), len(tt.wantNames))
			}
			for i, want := range tt.wantNames {
				if cmds[i].Name != want {
					t.Errorf("cmds[%d].Name: got %q want %q", i, cmds[i].Name, want)
				}
			}
		})
	}
}

func TestCommandsFromPostgres_multiDBName(t *testing.T) {
	tests := []struct {
		name      string
		dbname    string
		wantCount int
		wantDBs   []string
	}{
		{
			name:      "single dbname",
			dbname:    "hej",
			wantCount: 1,
			wantDBs:   []string{"hej"},
		},
		{
			name:      "comma separated dbnames",
			dbname:    "hej, hej1",
			wantCount: 2,
			wantDBs:   []string{"hej", "hej1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pg := &postgresdb.Postgres{Host: "localhost", Port: "5432", DBName: tc.dbname}
			cmds, err := CommandsFromPostgres(pg, "hba_scanner", nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(cmds) != 1 {
				t.Fatalf("cmds=%d want 1", len(cmds))
			}
			if len(cmds[0].Postgres) != tc.wantCount {
				t.Fatalf("postgres targets=%d want %d", len(cmds[0].Postgres), tc.wantCount)
			}
			for i, want := range tc.wantDBs {
				if cmds[0].Postgres[i].DBName != want {
					t.Fatalf("target[%d].DBName=%q want %q", i, cmds[0].Postgres[i].DBName, want)
				}
			}
		})
	}
}

func TestBuildCollectorScheduleMap(t *testing.T) {
	pg := &postgresdb.Postgres{Host: "localhost", Port: "5432", DBName: "hej"}
	logParser := &LogParserCronInput{
		Prefix:  "%t ",
		LogFile: "/var/log/postgresql.log",
	}

	tests := []struct {
		name       string
		cfg        *Config
		wantScheds map[string][]string
		wantErr    bool
	}{
		{
			name: "single schedule when optional schedules empty",
			cfg: &Config{
				Postgres: pg,
				Collector: Collector{
					Schedule:     "0 12 * * *",
					ScanCommands: "postgres_cis,pii_scanner,inactive_users",
					LogParser:    logParser,
				},
			},
			wantScheds: map[string][]string{
				"0 12 * * *": {cons.RootCMD_PostgresCIS, cons.RootCMD_PiiScanner, cons.LogParserCMD_InactiveUser},
			},
		},
		{
			name: "separate pii schedule",
			cfg: &Config{
				Postgres: pg,
				Collector: Collector{
					Schedule:     "0 12 * * *",
					ScanCommands: "postgres_cis,hba_scanner,pii_scanner",
				},
				PiiScanner: PiiScannerInput{Schedule: "0 3 * * 0"},
			},
			wantScheds: map[string][]string{
				"0 12 * * *": {cons.RootCMD_PostgresCIS, cons.RootCMD_HBAScanner},
				"0 3 * * 0":  {cons.RootCMD_PiiScanner},
			},
		},
		{
			name: "separate log parser schedule",
			cfg: &Config{
				Postgres: pg,
				Collector: Collector{
					Schedule:     "0 12 * * *",
					ScanCommands: "postgres_cis,inactive_users",
					LogParser: &LogParserCronInput{
						Schedule: "0 4 * * *",
						Prefix:   "%t ",
						LogFile:  "/var/log/postgresql.log",
					},
				},
			},
			wantScheds: map[string][]string{
				"0 12 * * *": {cons.RootCMD_PostgresCIS},
				"0 4 * * *":  {cons.LogParserCMD_InactiveUser},
			},
		},
		{
			name: "separate pii and log parser schedules",
			cfg: &Config{
				Postgres: pg,
				Collector: Collector{
					Schedule:     "0 12 * * *",
					ScanCommands: "postgres_cis,pii_scanner,inactive_users",
					LogParser: &LogParserCronInput{
						Schedule: "0 5 * * *",
						Prefix:   "%t ",
						LogFile:  "/var/log/postgresql.log",
					},
				},
				PiiScanner: PiiScannerInput{Schedule: "0 3 * * 0"},
			},
			wantScheds: map[string][]string{
				"0 12 * * *": {cons.RootCMD_PostgresCIS},
				"0 3 * * 0":  {cons.RootCMD_PiiScanner},
				"0 5 * * *":  {cons.LogParserCMD_InactiveUser},
			},
		},
		{
			name: "logparser schedule ignored without log parser scan command",
			cfg: &Config{
				Postgres: pg,
				Collector: Collector{
					Schedule:     "0 12 * * *",
					ScanCommands: "postgres_cis,hba_scanner",
					LogParser: &LogParserCronInput{
						Schedule: "0 4 * * *",
						LogFile:  "/var/log/postgresql.log",
					},
				},
			},
			wantScheds: map[string][]string{
				"0 12 * * *": {cons.RootCMD_PostgresCIS, cons.RootCMD_HBAScanner},
			},
		},
		{
			name: "all on single schedule when optional schedules empty",
			cfg: &Config{
				Postgres: pg,
				Collector: Collector{
					Schedule:     "0 12 * * *",
					ScanCommands: "all",
					LogParser: &LogParserCronInput{
						Prefix:      "%t ",
						LogFile:     "/var/log/postgresql.log",
						HbaConfFile: "/etc/pg_hba.conf",
					},
				},
			},
			wantScheds: map[string][]string{
				"0 12 * * *": {cons.RootCMD_All},
			},
		},
		{
			name: "all splits log parser to logparser schedule",
			cfg: &Config{
				Postgres: pg,
				Collector: Collector{
					Schedule:     "0 12 * * *",
					ScanCommands: "all",
					LogParser: &LogParserCronInput{
						Schedule:    "0 4 * * *",
						Prefix:      "%t ",
						LogFile:     "/var/log/postgresql.log",
						HbaConfFile: "/etc/pg_hba.conf",
					},
				},
			},
			wantScheds: map[string][]string{
				"0 12 * * *": {cons.RootCMD_AllCore, cons.RootCMD_PiiScanner},
				"0 4 * * *": {
					cons.LogParserCMD_InactiveUser,
					cons.LogParserCMD_UniqueIPs,
					cons.LogParserCMD_HBAUnusedLines,
					cons.LogParserCMD_PasswordLeakScanner,
				},
			},
		},
		{
			name: "all splits pii to piiscanner schedule",
			cfg: &Config{
				Postgres: pg,
				Collector: Collector{
					Schedule:     "0 12 * * *",
					ScanCommands: "all",
					LogParser: &LogParserCronInput{
						Prefix:      "%t ",
						LogFile:     "/var/log/postgresql.log",
						HbaConfFile: "/etc/pg_hba.conf",
					},
				},
				PiiScanner: PiiScannerInput{Schedule: "0 3 * * 0"},
			},
			wantScheds: map[string][]string{
				"0 12 * * *": {cons.RootCMD_AllCore, cons.LogParserCMD_InactiveUser, cons.LogParserCMD_UniqueIPs, cons.LogParserCMD_HBAUnusedLines, cons.LogParserCMD_PasswordLeakScanner},
				"0 3 * * 0":  {cons.RootCMD_PiiScanner},
			},
		},
		{
			name: "all splits pii and log parser to separate schedules",
			cfg: &Config{
				Postgres: pg,
				Collector: Collector{
					Schedule:     "0 12 * * *",
					ScanCommands: "all",
					LogParser: &LogParserCronInput{
						Schedule:    "0 4 * * *",
						Prefix:      "%t ",
						LogFile:     "/var/log/postgresql.log",
						HbaConfFile: "/etc/pg_hba.conf",
					},
				},
				PiiScanner: PiiScannerInput{Schedule: "0 3 * * 0"},
			},
			wantScheds: map[string][]string{
				"0 12 * * *": {cons.RootCMD_AllCore},
				"0 3 * * 0":  {cons.RootCMD_PiiScanner},
				"0 4 * * *": {
					cons.LogParserCMD_InactiveUser,
					cons.LogParserCMD_UniqueIPs,
					cons.LogParserCMD_HBAUnusedLines,
					cons.LogParserCMD_PasswordLeakScanner,
				},
			},
		},
		{
			name:       "not ready returns empty map",
			cfg:        &Config{Postgres: pg, Collector: Collector{ScanCommands: "postgres_cis"}},
			wantScheds: map[string][]string{},
		},
		{
			name: "legacy crons only",
			cfg: &Config{
				Crons: []Cron{{
					Schedule: "0 6 * * *",
					Commands: []Command{{Name: cons.RootCMD_PostgresCIS, Postgres: []*postgresdb.Postgres{pg}}},
				}},
			},
			wantScheds: map[string][]string{
				"0 6 * * *": {cons.RootCMD_PostgresCIS},
			},
		},
		{
			name: "legacy crons merged with collector schedule",
			cfg: &Config{
				Postgres: pg,
				Collector: Collector{
					Schedule:     "0 12 * * *",
					ScanCommands: "hba_scanner",
				},
				Crons: []Cron{{
					Schedule: "0 6 * * *",
					Commands: []Command{{Name: cons.RootCMD_PostgresCIS, Postgres: []*postgresdb.Postgres{pg}}},
				}},
			},
			wantScheds: map[string][]string{
				"0 12 * * *": {cons.RootCMD_HBAScanner},
				"0 6 * * *":  {cons.RootCMD_PostgresCIS},
			},
		},
		{
			name: "legacy crons reject sub-daily schedule",
			cfg: &Config{
				Crons: []Cron{{
					Schedule: "*/5 * * * *",
					Commands: []Command{{Name: cons.RootCMD_PostgresCIS, Postgres: []*postgresdb.Postgres{pg}}},
				}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.cfg.BuildCollectorScheduleMap()
			if (err != nil) != tt.wantErr {
				t.Fatalf("BuildCollectorScheduleMap() err=%v wantErr=%v", err, tt.wantErr)
			}
			if len(got) != len(tt.wantScheds) {
				t.Fatalf("schedules: got %d want %d (%v)", len(got), len(tt.wantScheds), got)
			}
			for sched, wantNames := range tt.wantScheds {
				cmds, ok := got[sched]
				if !ok {
					t.Fatalf("missing schedule %q", sched)
				}
				if len(cmds) != len(wantNames) {
					t.Fatalf("schedule %q: got %d commands want %d", sched, len(cmds), len(wantNames))
				}
				for i, want := range wantNames {
					if cmds[i].Name != want {
						t.Fatalf("schedule %q cmds[%d]: got %q want %q", sched, i, cmds[i].Name, want)
					}
				}
			}
		})
	}
}
