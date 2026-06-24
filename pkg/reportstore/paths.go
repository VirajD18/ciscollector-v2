package reportstore

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/klouddb/klouddbshield/pkg/config"
)

// DefaultStorage returns defaults when [storage] is omitted from TOML.
func DefaultStorage() config.StorageConfig {
	return config.StorageConfig{
		Enabled:          true,
		BasePath:         "~/.klouddb",
		SQLiteFile:       "klouddbshield.db",
		RetentionDays:    90,
		RetentionPurgeOn: "cron_tick",
	}
}

// ResolveDBPath returns absolute path to the dashboard SQLite file.
func ResolveDBPath(s config.StorageConfig) (string, error) {
	if s.BasePath == "" {
		s.BasePath = DefaultStorage().BasePath
	}
	if s.SQLiteFile == "" {
		s.SQLiteFile = DefaultStorage().SQLiteFile
	}
	base := s.BasePath
	if strings.HasPrefix(base, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, strings.TrimPrefix(base, "~/"))
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(base, s.SQLiteFile), nil
}
