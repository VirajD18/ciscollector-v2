package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// DefaultOfflineThresholdSec is how long without heartbeat before a collector is offline (1.5 min).
const DefaultOfflineThresholdSec = 90

// LoadFromPath reads kshieldconfig.toml from a file path or directory (same as -config flag).
func LoadFromPath(path string) (*Config, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("config path is empty")
	}
	if st, err := os.Stat(path); err == nil && st.IsDir() {
		return LoadConfig(path)
	}
	v := viper.New()
	v.SetConfigType("toml")
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	c := &Config{}
	if err := v.Unmarshal(c, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "toml"
	}); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	c.StorageSet = v.IsSet("storage")
	return c, nil
}

// OfflineThresholdFromPath returns mainserver.offline_threshold_sec from config (default 90s).
func OfflineThresholdFromPath(path string) int {
	c, err := LoadFromPath(path)
	if err != nil || c == nil {
		return DefaultOfflineThresholdSec
	}
	sec := c.MainServer.OfflineThresholdSec
	if sec <= 0 {
		return DefaultOfflineThresholdSec
	}
	return sec
}
