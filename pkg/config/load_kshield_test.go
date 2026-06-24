package config

import (
	"path/filepath"
	"testing"
)

func TestLoadProjectKshieldConfig(t *testing.T) {
	root := filepath.Join("..", "..")
	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	t.Logf("Collector.Schedule=%q ScanCommands=%q LogParser=%v", cfg.Collector.Schedule, cfg.Collector.ScanCommands, cfg.Collector.LogParser)
	if !cfg.CollectorScheduleReady() {
		t.Fatalf("expected collector schedule ready; scan_commands=%q", cfg.Collector.ScanCommands)
	}
	if cfg.Collector.ScanCommands != "all" {
		t.Fatalf("scan_commands=%q want all", cfg.Collector.ScanCommands)
	}
}
