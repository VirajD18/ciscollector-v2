package main

import (
	"os"
	"strings"

	"github.com/klouddb/klouddbshield/pkg/repository"
)

// resolveStorageConfig merges CLI values, server-node.yaml, and environment into storage config.
// Precedence (highest first): environment variables, CLI flags, server-node.yaml.
func resolveStorageConfig(cli repository.Config, node ServerConfig) repository.Config {
	cfg := cli

	if strings.TrimSpace(cfg.PostgresURL) == "" {
		cfg.PostgresURL = strings.TrimSpace(node.PostgresURL)
	}

	if shouldApplyServerNodeDriver(cfg, node) {
		cfg.Driver = strings.TrimSpace(node.DBDriver)
	}

	if cfg.EffectiveDriver() == repository.DriverSQLite && strings.TrimSpace(cfg.PostgresURL) != "" {
		cfg.Driver = repository.DriverPostgres
	}

	return repository.ResolveFromEnv(cfg)
}

func shouldApplyServerNodeDriver(cfg repository.Config, node ServerConfig) bool {
	if strings.TrimSpace(node.DBDriver) == "" {
		return false
	}
	if strings.TrimSpace(os.Getenv("KSHIELD_DB_DRIVER")) != "" {
		return false
	}
	return cfg.EffectiveDriver() != repository.DriverPostgres
}
