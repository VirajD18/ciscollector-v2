package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEffectiveOfflineThreshold(t *testing.T) {
	tests := []struct {
		name string
		ms   MainServer
		want time.Duration
	}{
		{
			name: "default_90s",
			ms:   MainServer{},
			want: 90 * time.Second,
		},
		{
			name: "custom_120s",
			ms:   MainServer{OfflineThresholdSec: 120},
			want: 120 * time.Second,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.ms.EffectiveOfflineThreshold(); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestOfflineThresholdFromPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kshieldconfig.toml")
	if err := writeTestConfig(path, `[mainserver]
offline_threshold_sec = 75
`); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		path string
		want int
	}{
		{name: "from_file", path: path, want: 75},
		{name: "missing_defaults", path: filepath.Join(dir, "missing.toml"), want: DefaultOfflineThresholdSec},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := OfflineThresholdFromPath(tc.path); got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}

func writeTestConfig(path, body string) error {
	return os.WriteFile(path, []byte(body), 0o600)
}
