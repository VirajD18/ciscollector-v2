package config

import (
	"fmt"
	"strings"

	cons "github.com/VirajD18/ciscollector-v2/pkg/const"
	"github.com/VirajD18/ciscollector-v2/pkg/piiscanner"
	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
)

// PiiScannerInput holds [piiscanner] settings from kshieldconfig.toml.
type PiiScannerInput struct {
	Schedule      string `toml:"schedule"`
	RunOption     string `toml:"run_option"`
	Database      string `toml:"database"`
	Schema        string `toml:"schema"`
	ExcludeTables string `toml:"exclude_tables"`
	IncludeTables string `toml:"include_tables"`
}

// PiiScannerOverrides supplies CLI flag values that override TOML when non-empty.
type PiiScannerOverrides struct {
	RunOption     string
	Database      string
	Schema        string
	ExcludeTables string
	IncludeTables string
	PrintAll      bool
	SpacyOnly     bool
	PrintSummary  bool
}

func (in PiiScannerInput) merged(ov PiiScannerOverrides) (runOption, database, schema, exclude, include string) {
	runOption = strings.TrimSpace(in.RunOption)
	if s := strings.TrimSpace(ov.RunOption); s != "" {
		runOption = s
	}
	if runOption == "" {
		runOption = piiscanner.RunOption_DataScan_String
	}

	database = strings.TrimSpace(in.Database)
	if s := strings.TrimSpace(ov.Database); s != "" {
		database = s
	}

	schema = strings.TrimSpace(in.Schema)
	if schema == "" {
		schema = "public"
	}
	if s := strings.TrimSpace(ov.Schema); s != "" {
		schema = s
	}

	exclude = strings.TrimSpace(in.ExcludeTables)
	if s := strings.TrimSpace(ov.ExcludeTables); s != "" {
		exclude = s
	}

	include = strings.TrimSpace(in.IncludeTables)
	if s := strings.TrimSpace(ov.IncludeTables); s != "" {
		include = s
	}
	return
}

// BuildPiiScannerConfig merges [piiscanner] TOML with optional CLI overrides.
func (c *Config) BuildPiiScannerConfig(pg *postgresdb.Postgres, ov PiiScannerOverrides) (*piiscanner.Config, error) {
	if pg == nil {
		return nil, fmt.Errorf(cons.Err_PostgresConfig_Missing)
	}
	if c == nil {
		return nil, fmt.Errorf("config is nil")
	}
	runOption, database, schema, exclude, include := c.PiiScanner.merged(ov)
	return piiscanner.NewConfig(pg, runOption, exclude, include, database, schema,
		ov.PrintAll, ov.SpacyOnly, ov.PrintSummary)
}
