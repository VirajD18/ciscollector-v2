package repository_test

import (
	"testing"

	"github.com/VirajD18/ciscollector-v2/pkg/repository"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     repository.Config
		wantErr bool
	}{
		{
			name: "sqlite default path",
			cfg: repository.Config{
				Driver:     repository.DriverSQLite,
				SQLitePath: "/tmp/main-server.sqlite",
			},
		},
		{
			name: "sqlite missing path",
			cfg: repository.Config{
				Driver: repository.DriverSQLite,
			},
			wantErr: true,
		},
		{
			name: "postgres with url",
			cfg: repository.Config{
				Driver:      repository.DriverPostgres,
				PostgresURL: "postgres://user:pass@localhost:5432/shield?sslmode=disable",
			},
		},
		{
			name: "postgres missing url",
			cfg: repository.Config{
				Driver: repository.DriverPostgres,
			},
			wantErr: true,
		},
		{
			name: "driver alias pg",
			cfg: repository.Config{
				Driver:      "pg",
				PostgresURL: "postgres://localhost/shield",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEffectiveDriver(t *testing.T) {
	tests := []struct {
		driver string
		want   string
	}{
		{driver: "", want: repository.DriverSQLite},
		{driver: "sqlite", want: repository.DriverSQLite},
		{driver: "postgres", want: repository.DriverPostgres},
		{driver: "postgresql", want: repository.DriverPostgres},
		{driver: "pg", want: repository.DriverPostgres},
	}

	for _, tt := range tests {
		t.Run(tt.driver, func(t *testing.T) {
			got := repository.Config{Driver: tt.driver}.EffectiveDriver()
			if got != tt.want {
				t.Fatalf("EffectiveDriver() = %q, want %q", got, tt.want)
			}
		})
	}
}
