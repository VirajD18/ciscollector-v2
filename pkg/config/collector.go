package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Collector holds top-level default schedule and scan commands for single-host [postgres] targets.
type Collector struct {
	Schedule     string              `toml:"schedule"`
	ScanCommands string              `toml:"scan_commands"`
	LogParser    *LogParserCronInput `toml:"logparser"`
}

// MainServer configures push from ciscollector to the dashboard main-server.
type MainServer struct {
	Enabled             bool   `toml:"enabled"`
	URL                 string `toml:"url"`
	Token               string `toml:"token"`
	PushInterval        string `toml:"push_interval"`
	OfflineThresholdSec int    `toml:"offline_threshold_sec"`
	// TLSCAFile pins trust to a PEM bundle (corporate/internal PKI). Omit for public CAs.
	TLSCAFile string `toml:"tls_ca_file"`
}

// EffectivePushInterval returns the heartbeat interval (default 30s).
func (m MainServer) EffectivePushInterval() time.Duration {
	raw := strings.TrimSpace(m.PushInterval)
	if raw == "" {
		return 30 * time.Second
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 30 * time.Second
	}
	return d
}

// EffectiveOfflineThreshold returns how long without heartbeat counts as offline (default 90s / 1.5 min).
func (m MainServer) EffectiveOfflineThreshold() time.Duration {
	if m.OfflineThresholdSec <= 0 {
		return DefaultOfflineThresholdSec * time.Second
	}
	return time.Duration(m.OfflineThresholdSec) * time.Second
}

// Validate returns an error when main server push is enabled but misconfigured.
func (m MainServer) Validate() error {
	if !m.Enabled {
		return nil
	}
	if strings.TrimSpace(m.URL) == "" {
		return fmt.Errorf("mainserver.url is required when mainserver.enabled is true")
	}
	if strings.TrimSpace(m.Token) == "" {
		return fmt.Errorf("mainserver.token is required when mainserver.enabled is true")
	}
	if ca := strings.TrimSpace(m.TLSCAFile); ca != "" {
		if _, err := os.Stat(ca); err != nil {
			return fmt.Errorf("mainserver.tls_ca_file: %w", err)
		}
	}
	return nil
}

// CollectorScheduleReady is true when single-host collector scheduling can run.
func (c *Config) CollectorScheduleReady() bool {
	col := c.Collector
	return c.Postgres != nil &&
		strings.TrimSpace(col.Schedule) != "" &&
		strings.TrimSpace(col.ScanCommands) != ""
}

// DefaultLogPrefix is the PostgreSQL log_line_prefix token sequence for logparser.
func DefaultLogPrefix() string {
	return "%t "
}
