package postgresdb

import (
	"strings"
	"testing"
)

func TestPostgresExpandTargets(t *testing.T) {
	tests := []struct {
		name    string
		dbname  string
		wantDBs []string
	}{
		{name: "single", dbname: "hej", wantDBs: []string{"hej"}},
		{name: "comma separated", dbname: "hej, hej1", wantDBs: []string{"hej", "hej1"}},
		{name: "extra spaces", dbname: " hej , hej1 , hej2 ", wantDBs: []string{"hej", "hej1", "hej2"}},
		{name: "empty", dbname: "  , ", wantDBs: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := (&Postgres{DBName: tc.dbname}).ExpandTargets()
			if len(tc.wantDBs) == 0 {
				if len(got) != 0 {
					t.Fatalf("ExpandTargets()=%d want empty", len(got))
				}
				return
			}
			if len(got) != len(tc.wantDBs) {
				t.Fatalf("ExpandTargets() count=%d want %d", len(got), len(tc.wantDBs))
			}
			for i, want := range tc.wantDBs {
				if got[i].DBName != want {
					t.Fatalf("target[%d].DBName=%q want %q", i, got[i].DBName, want)
				}
			}
		})
	}
}

func TestPostgresValidate(t *testing.T) {
	tests := []struct {
		name          string
		config        *Postgres
		wantErr       bool
		wantMissing   []string
		wantErrSubstr string
	}{
		{
			name: "all_present",
			config: &Postgres{
				Host: "localhost", Port: "5432", User: "postgres",
				Password: "secret", DBName: "hej",
			},
			wantErr: false,
		},
		{
			name:    "comma separated dbnames",
			config:  &Postgres{Host: "localhost", Port: "5432", User: "postgres", Password: "secret", DBName: "hej, hej1"},
			wantErr: false,
		},
		{
			name:          "nil_config",
			config:        nil,
			wantErr:       true,
			wantMissing:   []string{"host", "port", "user", "password", "dbname"},
			wantErrSubstr: "missing [postgres] host, port, user, password, dbname",
		},
		{
			name:          "missing_host_and_dbname",
			config:        &Postgres{Port: "5432", User: "postgres", Password: "secret"},
			wantErr:       true,
			wantMissing:   []string{"host", "dbname"},
			wantErrSubstr: "missing [postgres] host, dbname",
		},
		{
			name:          "missing_password_only",
			config:        &Postgres{Host: "localhost", Port: "5432", User: "postgres", DBName: "hej"},
			wantErr:       true,
			wantMissing:   []string{"password"},
			wantErrSubstr: "missing [postgres] password",
		},
		{
			name:          "whitespace_only_fields",
			config:        &Postgres{Host: "  ", Port: "\t", User: "postgres", Password: "x", DBName: "hej"},
			wantErr:       true,
			wantMissing:   []string{"host", "port"},
			wantErrSubstr: "missing [postgres] host, port",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotMissing := tc.config.MissingRequiredFields()
			if len(gotMissing) != len(tc.wantMissing) {
				t.Fatalf("MissingRequiredFields()=%v want %v", gotMissing, tc.wantMissing)
			}
			for i := range tc.wantMissing {
				if gotMissing[i] != tc.wantMissing[i] {
					t.Fatalf("MissingRequiredFields()=%v want %v", gotMissing, tc.wantMissing)
				}
			}

			err := tc.config.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() err=%v wantErr=%v", err, tc.wantErr)
			}
			if tc.wantErr && !strings.Contains(err.Error(), tc.wantErrSubstr) {
				t.Fatalf("Validate()=%q want substring %q", err.Error(), tc.wantErrSubstr)
			}
		})
	}
}

func TestBuildConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		config   Postgres
		expected string
	}{
		{
			name: "Basic connection without SSL",
			config: Postgres{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "password",
				DBName:   "testdb",
			},
			expected: "host=localhost port=5432 user=postgres password=password dbname=testdb sslmode=disable",
		},
		{
			name: "Connection with SSL mode enabled",
			config: Postgres{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "password",
				DBName:   "testdb",
				SSLmode:  "require",
			},
			expected: "host=localhost port=5432 user=postgres password=password dbname=testdb sslmode=require",
		},
		{
			name: "Connection with SSL certificates",
			config: Postgres{
				Host:        "localhost",
				Port:        "5432",
				User:        "postgres",
				Password:    "password",
				DBName:      "testdb",
				SSLmode:     "verify-full",
				SSLcert:     "/path/to/client.crt",
				SSLkey:      "/path/to/client.key",
				SSLrootcert: "/path/to/root.crt",
			},
			expected: "host=localhost port=5432 user=postgres password=password dbname=testdb sslmode=verify-full sslcert=/path/to/client.crt sslkey=/path/to/client.key sslrootcert=/path/to/root.crt",
		},
		{
			name: "Connection with partial SSL certificates",
			config: Postgres{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "password",
				DBName:   "testdb",
				SSLmode:  "require",
				SSLcert:  "/path/to/client.crt",
				SSLkey:   "/path/to/client.key",
			},
			expected: "host=localhost port=5432 user=postgres password=password dbname=testdb sslmode=require sslcert=/path/to/client.crt sslkey=/path/to/client.key",
		},
		{
			name: "Connection with empty SSL mode defaults to disable",
			config: Postgres{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "password",
				DBName:   "testdb",
				SSLmode:  "",
			},
			expected: "host=localhost port=5432 user=postgres password=password dbname=testdb sslmode=disable",
		},
		{
			name: "Connection with SSL certificates but no SSL mode",
			config: Postgres{
				Host:        "localhost",
				Port:        "5432",
				User:        "postgres",
				Password:    "password",
				DBName:      "testdb",
				SSLcert:     "/path/to/client.crt",
				SSLkey:      "/path/to/client.key",
				SSLrootcert: "/path/to/root.crt",
			},
			expected: "host=localhost port=5432 user=postgres password=password dbname=testdb sslmode=disable sslcert=/path/to/client.crt sslkey=/path/to/client.key sslrootcert=/path/to/root.crt",
		},
		{
			name: "Connection with special characters in password",
			config: Postgres{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "pass@word#123",
				DBName:   "testdb",
				SSLmode:  "require",
			},
			expected: "host=localhost port=5432 user=postgres password=pass@word#123 dbname=testdb sslmode=require",
		},
		{
			name: "Connection with IPv6 host",
			config: Postgres{
				Host:     "::1",
				Port:     "5432",
				User:     "postgres",
				Password: "password",
				DBName:   "testdb",
				SSLmode:  "require",
			},
			expected: "host=::1 port=5432 user=postgres password=password dbname=testdb sslmode=require",
		},
		{
			name: "Connection with custom port",
			config: Postgres{
				Host:     "localhost",
				Port:     "5433",
				User:     "postgres",
				Password: "password",
				DBName:   "testdb",
				SSLmode:  "require",
			},
			expected: "host=localhost port=5433 user=postgres password=password dbname=testdb sslmode=require",
		},
		{
			name: "Connection with all SSL modes",
			config: Postgres{
				Host:        "localhost",
				Port:        "5432",
				User:        "postgres",
				Password:    "password",
				DBName:      "testdb",
				SSLmode:     "prefer",
				SSLcert:     "/path/to/client.crt",
				SSLkey:      "/path/to/client.key",
				SSLrootcert: "/path/to/root.crt",
			},
			expected: "host=localhost port=5432 user=postgres password=password dbname=testdb sslmode=prefer sslcert=/path/to/client.crt sslkey=/path/to/client.key sslrootcert=/path/to/root.crt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildConnectionString(tt.config)
			if result != tt.expected {
				t.Errorf("BuildConnectionString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuildConnectionString_SSLModeVariations(t *testing.T) {
	baseConfig := Postgres{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "password",
		DBName:   "testdb",
	}

	sslModes := []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}

	for _, mode := range sslModes {
		t.Run("SSL mode: "+mode, func(t *testing.T) {
			config := baseConfig
			config.SSLmode = mode

			result := BuildConnectionString(config)
			expected := "host=localhost port=5432 user=postgres password=password dbname=testdb sslmode=" + mode

			if result != expected {
				t.Errorf("BuildConnectionString() with sslmode=%s = %v, want %v", mode, result, expected)
			}
		})
	}
}

func TestBuildConnectionString_EmptyValues(t *testing.T) {
	tests := []struct {
		name     string
		config   Postgres
		expected string
	}{
		{
			name: "Empty SSL certificates should not be included",
			config: Postgres{
				Host:        "localhost",
				Port:        "5432",
				User:        "postgres",
				Password:    "password",
				DBName:      "testdb",
				SSLmode:     "require",
				SSLcert:     "",
				SSLkey:      "",
				SSLrootcert: "",
			},
			expected: "host=localhost port=5432 user=postgres password=password dbname=testdb sslmode=require",
		},
		{
			name: "Mixed empty and non-empty SSL certificates",
			config: Postgres{
				Host:        "localhost",
				Port:        "5432",
				User:        "postgres",
				Password:    "password",
				DBName:      "testdb",
				SSLmode:     "require",
				SSLcert:     "/path/to/client.crt",
				SSLkey:      "",
				SSLrootcert: "/path/to/root.crt",
			},
			expected: "host=localhost port=5432 user=postgres password=password dbname=testdb sslmode=require sslcert=/path/to/client.crt sslrootcert=/path/to/root.crt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildConnectionString(tt.config)
			if result != tt.expected {
				t.Errorf("BuildConnectionString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuildConnectionString_OrderConsistency(t *testing.T) {
	// Test that the connection string parameters are always in the same order
	config := Postgres{
		Host:        "localhost",
		Port:        "5432",
		User:        "postgres",
		Password:    "password",
		DBName:      "testdb",
		SSLmode:     "require",
		SSLcert:     "/path/to/client.crt",
		SSLkey:      "/path/to/client.key",
		SSLrootcert: "/path/to/root.crt",
	}

	// Run the function multiple times to ensure consistent ordering
	results := make([]string, 5)
	for i := 0; i < 5; i++ {
		results[i] = BuildConnectionString(config)
	}

	// All results should be identical
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("BuildConnectionString() returned inconsistent results: %v vs %v", results[0], results[i])
		}
	}
}

func TestBuildConnectionString_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name     string
		config   Postgres
		expected string
	}{
		{
			name: "Production-like configuration",
			config: Postgres{
				Host:        "prod-db.example.com",
				Port:        "5432",
				User:        "app_user",
				Password:    "secure_password_123",
				DBName:      "production_db",
				SSLmode:     "verify-full",
				SSLcert:     "/etc/ssl/certs/client.crt",
				SSLkey:      "/etc/ssl/private/client.key",
				SSLrootcert: "/etc/ssl/certs/ca-bundle.crt",
			},
			expected: "host=prod-db.example.com port=5432 user=app_user password=secure_password_123 dbname=production_db sslmode=verify-full sslcert=/etc/ssl/certs/client.crt sslkey=/etc/ssl/private/client.key sslrootcert=/etc/ssl/certs/ca-bundle.crt",
		},
		{
			name: "Development configuration",
			config: Postgres{
				Host:     "localhost",
				Port:     "5432",
				User:     "dev_user",
				Password: "dev_password",
				DBName:   "dev_db",
				SSLmode:  "disable",
			},
			expected: "host=localhost port=5432 user=dev_user password=dev_password dbname=dev_db sslmode=disable",
		},
		{
			name: "Docker container configuration",
			config: Postgres{
				Host:     "postgres-container",
				Port:     "5432",
				User:     "postgres",
				Password: "docker_password",
				DBName:   "app_db",
				SSLmode:  "prefer",
			},
			expected: "host=postgres-container port=5432 user=postgres password=docker_password dbname=app_db sslmode=prefer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildConnectionString(tt.config)
			if result != tt.expected {
				t.Errorf("BuildConnectionString() = %v, want %v", result, tt.expected)
			}
		})
	}
}
