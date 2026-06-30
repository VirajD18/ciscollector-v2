package service

import (
	"os"
	"path/filepath"

	"github.com/VirajD18/ciscollector-v2/pkg/config"
	"github.com/pelletier/go-toml/v2"
)

// ResolveKshieldConfigPath finds kshieldconfig.toml for dashboard admin APIs.
func ResolveKshieldConfigPath(explicit string) string {
	if explicit != "" {
		if _, err := os.Stat(explicit); err == nil {
			return explicit
		}
	}
	if p := os.Getenv("KSHIELD_CONFIG"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	candidates := []string{
		"kshieldconfig.toml",
		filepath.Join(".", "kshieldconfig.toml"),
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "kshieldconfig.toml"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, "kshieldconfig.toml"),
			filepath.Join(home, "Desktop", "akshitkshieldapptestversion", "kshieldconfig.toml"),
		)
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func loadKshieldConfig(path string) (*config.Config, error) {
	if path == "" {
		return nil, os.ErrNotExist
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c config.Config
	if err := toml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
