package repository

import (
	"fmt"
	"os"
	"strings"
)

const (
	DriverSQLite   = "sqlite"
	DriverPostgres = "postgres"
)

// Config selects the main-server storage backend.
type Config struct {
	Driver      string
	SQLitePath  string
	PostgresURL string
}

// EffectiveDriver returns a normalized driver name (defaults to sqlite).
func (c Config) EffectiveDriver() string {
	switch strings.ToLower(strings.TrimSpace(c.Driver)) {
	case DriverPostgres, "postgresql", "pg":
		return DriverPostgres
	default:
		return DriverSQLite
	}
}

// ResolveFromEnv merges CLI/env overrides into cfg.
// Environment variables take precedence over values from CLI flags or server-node.yaml.
func ResolveFromEnv(cfg Config) Config {
	if v := strings.TrimSpace(os.Getenv("KSHIELD_DB_DRIVER")); v != "" {
		cfg.Driver = v
	}
	if v := strings.TrimSpace(os.Getenv("DATABASE_URL")); v != "" {
		cfg.PostgresURL = v
	} else if v := strings.TrimSpace(os.Getenv("MAIN_SERVER_DATABASE_URL")); v != "" {
		cfg.PostgresURL = v
	}
	return cfg
}

// Validate checks required fields for the selected driver.
func (c Config) Validate() error {
	switch c.EffectiveDriver() {
	case DriverPostgres:
		if strings.TrimSpace(c.PostgresURL) == "" {
			return fmt.Errorf("postgres driver requires postgres_url or DATABASE_URL")
		}
	case DriverSQLite:
		if strings.TrimSpace(c.SQLitePath) == "" {
			return fmt.Errorf("sqlite driver requires sqlite_path")
		}
	}
	return nil
}
