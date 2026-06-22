package main

import (
	"testing"

	"github.com/klouddb/klouddbshield/pkg/repository"
)

func TestResolveStorageConfig(t *testing.T) {
	const pgURL = "postgres://kshield:kshield@localhost:5432/kshield?sslmode=disable"

	tests := []struct {
		name     string
		cli      repository.Config
		node     ServerConfig
		env      map[string]string
		wantDrv  string
		wantPG   string
	}{
		{
			name:    "cli sqlite defaults use server-node postgres",
			cli:     repository.Config{Driver: "sqlite"},
			node:    ServerConfig{DBDriver: "postgres", PostgresURL: pgURL},
			wantDrv: repository.DriverPostgres,
			wantPG:  pgURL,
		},
		{
			name:    "server-node postgres_url alone implies postgres driver",
			cli:     repository.Config{Driver: "sqlite"},
			node:    ServerConfig{PostgresURL: pgURL},
			wantDrv: repository.DriverPostgres,
			wantPG:  pgURL,
		},
		{
			name:    "cli postgres driver kept over server-node sqlite",
			cli:     repository.Config{Driver: "postgres", PostgresURL: pgURL},
			node:    ServerConfig{DBDriver: "sqlite"},
			wantDrv: repository.DriverPostgres,
			wantPG:  pgURL,
		},
		{
			name:    "env driver overrides server-node",
			cli:     repository.Config{Driver: "sqlite"},
			node:    ServerConfig{DBDriver: "postgres", PostgresURL: pgURL},
			env:     map[string]string{"KSHIELD_DB_DRIVER": "sqlite"},
			wantDrv: repository.DriverSQLite,
			wantPG:  pgURL,
		},
		{
			name:    "env database url overrides server-node",
			cli:     repository.Config{Driver: "sqlite"},
			node:    ServerConfig{DBDriver: "postgres", PostgresURL: pgURL},
			env:     map[string]string{"DATABASE_URL": "postgres://env:env@db:5432/app?sslmode=disable"},
			wantDrv: repository.DriverPostgres,
			wantPG:  "postgres://env:env@db:5432/app?sslmode=disable",
		},
		{
			name:    "main server database url env supported",
			cli:     repository.Config{Driver: "sqlite"},
			node:    ServerConfig{DBDriver: "postgres"},
			env: map[string]string{
				"KSHIELD_DB_DRIVER":          "postgres",
				"MAIN_SERVER_DATABASE_URL": pgURL,
			},
			wantDrv: repository.DriverPostgres,
			wantPG:  pgURL,
		},
		{
			name:    "cli postgres url kept when server-node has different url",
			cli:     repository.Config{Driver: "postgres", PostgresURL: "postgres://cli:cli@db:5432/app?sslmode=disable"},
			node:    ServerConfig{PostgresURL: pgURL},
			wantDrv: repository.DriverPostgres,
			wantPG:  "postgres://cli:cli@db:5432/app?sslmode=disable",
		},
		{
			name:    "sqlite when no server-node storage fields",
			cli:     repository.Config{Driver: "sqlite", SQLitePath: "/tmp/main-server.sqlite"},
			node:    ServerConfig{},
			wantDrv: repository.DriverSQLite,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			for _, k := range []string{"KSHIELD_DB_DRIVER", "DATABASE_URL", "MAIN_SERVER_DATABASE_URL"} {
				if _, ok := tt.env[k]; !ok {
					t.Setenv(k, "")
				}
			}

			got := resolveStorageConfig(tt.cli, tt.node)
			if got.EffectiveDriver() != tt.wantDrv {
				t.Fatalf("driver: got %q want %q", got.EffectiveDriver(), tt.wantDrv)
			}
			if tt.wantPG != "" && got.PostgresURL != tt.wantPG {
				t.Fatalf("postgres url: got %q want %q", got.PostgresURL, tt.wantPG)
			}
		})
	}
}

func TestShouldApplyServerNodeDriver(t *testing.T) {
	tests := []struct {
		name string
		cli  repository.Config
		node ServerConfig
		env  string
		want bool
	}{
		{
			name: "apply when cli sqlite and node postgres",
			cli:  repository.Config{Driver: "sqlite"},
			node: ServerConfig{DBDriver: "postgres"},
			want: true,
		},
		{
			name: "skip when cli postgres",
			cli:  repository.Config{Driver: "postgres", PostgresURL: "postgres://u:p@h/db"},
			node: ServerConfig{DBDriver: "sqlite"},
			want: false,
		},
		{
			name: "skip when env sets driver",
			cli:  repository.Config{Driver: "sqlite"},
			node: ServerConfig{DBDriver: "postgres"},
			env:  "sqlite",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != "" {
				t.Setenv("KSHIELD_DB_DRIVER", tt.env)
			} else {
				t.Setenv("KSHIELD_DB_DRIVER", "")
			}

			got := shouldApplyServerNodeDriver(tt.cli, tt.node)
			if got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}
