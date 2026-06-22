package service

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
)

// CollectorConfig exposes scan features and crons from config + SQLite feature history.
func (s *Service) CollectorConfig(ctx context.Context) (*CollectorConfigResponse, error) {
	featureCatalog := []CollectorFeature{
		{ID: "postgres_cis", Label: "CIS benchmarks", Enabled: false, Menu: "2"},
		{ID: "hba_scanner", Label: "HBA scanner", Enabled: false, Menu: "3"},
		{ID: "pii_scan", Label: "PII scan", Enabled: false, Menu: "4"},
		{ID: "log_parser", Label: "Log parser", Enabled: false, Menu: "6-10"},
		{ID: "password_leak", Label: "Password leakage", Enabled: false, Menu: "10"},
		{ID: "config_audit", Label: "Config audit", Enabled: false, Menu: "16"},
		{ID: "ssl_audit", Label: "SSL audit", Enabled: false, Menu: "15"},
	}
	enabled := map[string]bool{}
	if cfg, err := loadKshieldConfig(ResolveKshieldConfigPath(s.ConfigPath)); err == nil && cfg != nil {
		for _, cr := range cfg.Crons {
			for _, cmd := range cr.Commands {
				enabled[strings.ToLower(cmd.Name)] = true
				switch strings.ToLower(cmd.Name) {
				case "all":
					for i := range featureCatalog {
						featureCatalog[i].Enabled = true
					}
				case "postgres_cis", "hba_scanner", "ssl_check", "ssl_audit", "inactive_users", "unique_ip", "password_leak_scanner":
					id := strings.ToLower(cmd.Name)
					if id == "password_leak_scanner" {
						id = "password_leak"
					}
					if id == "ssl_check" {
						id = "ssl_audit"
					}
					enabled[id] = true
				}
			}
		}
	}
	runs, _ := s.latestRunsByTarget(ctx)
	for _, run := range runs {
		for _, f := range run.FeaturesRun {
			id := strings.ToLower(f)
			enabled[id] = true
			if id == "ssl_check" {
				enabled["ssl_audit"] = true
			}
		}
	}
	for i := range featureCatalog {
		if enabled[featureCatalog[i].ID] {
			featureCatalog[i].Enabled = true
		}
	}
	crons := []NotificationSchedule{}
	if cfg, err := loadKshieldConfig(ResolveKshieldConfigPath(s.ConfigPath)); err == nil && cfg != nil {
		if schedMap, err := cfg.BuildCollectorScheduleMap(); err == nil {
			schedules := make([]string, 0, len(schedMap))
			for sched := range schedMap {
				schedules = append(schedules, sched)
			}
			sort.Strings(schedules)
			for i, sched := range schedules {
				cmds := schedMap[sched]
				features := make([]string, 0, len(cmds))
				for _, cmd := range cmds {
					features = append(features, cmd.Name)
				}
				crons = append(crons, NotificationSchedule{
					Name:     fmt.Sprintf("schedule_%d", i+1),
					Cron:     sched,
					Features: strings.Join(features, ", "),
				})
			}
		}
	}
	if len(crons) == 0 {
		crons = []NotificationSchedule{{Name: "none", Cron: "-", Features: "Set [collector] schedule and scan_commands or [[crons]] in kshieldconfig.toml"}}
	}
	return &CollectorConfigResponse{Features: featureCatalog, Crons: crons}, nil
}

// ConfigPathFromEnv resolves collector config path for main-server startup.
func ConfigPathFromEnv() string {
	if p := os.Getenv("KSHIELD_CONFIG"); p != "" {
		return p
	}
	if _, err := os.Stat("kshieldconfig.toml"); err == nil {
		wd, _ := os.Getwd()
		if wd != "" {
			return wd + string(os.PathSeparator) + "kshieldconfig.toml"
		}
		return "kshieldconfig.toml"
	}
	return ""
}
