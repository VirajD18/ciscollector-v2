package config

// StorageConfig controls dashboard SQLite persistence (~/.klouddb/klouddbshield.db).
type StorageConfig struct {
	Enabled          bool   `toml:"enabled"`
	BasePath         string `toml:"base_path"`
	SQLiteFile       string `toml:"sqlite_file"`
	RetentionDays    int    `toml:"retention_days"`
	RetentionPurgeOn string `toml:"retention_purge_on"` // cron_tick | daily | both
}

// DatabaseConfig selects the main-server storage backend.
type DatabaseConfig struct {
	Driver      string `toml:"driver"`       // sqlite | postgres
	SQLitePath  string `toml:"sqlite_path"`  // optional override for main-server.sqlite
	PostgresURL string `toml:"postgres_url"` // postgres connection string
}

// EffectiveStorage returns storage settings with defaults applied.
func (c *Config) EffectiveStorage() StorageConfig {
	s := c.Storage
	if s.BasePath == "" {
		s.BasePath = "~/.klouddb"
	}
	if s.SQLiteFile == "" {
		s.SQLiteFile = "klouddbshield.db"
	}
	if s.RetentionDays == 0 {
		s.RetentionDays = 90
	}
	if s.RetentionPurgeOn == "" {
		s.RetentionPurgeOn = "cron_tick"
	}
	if !c.StorageSet {
		s.Enabled = false
	}
	return s
}
