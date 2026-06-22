package postgresdb

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

type Postgres struct {
	Host        string `toml:"host"`
	Port        string `toml:"port"`
	User        string `toml:"user"`
	Password    string `toml:"password"`
	DBName      string `toml:"dbname"`
	SSLmode     string `toml:"sslmode"`
	SSLcert     string `toml:"sslcert"`
	SSLkey      string `toml:"sslkey"`
	SSLrootcert string `toml:"sslrootcert"`
	PingCheck   bool   `toml:"pingCheck"`
	MaxIdleConn int    `toml:"maxIdleConn"`
	MaxOpenConn int    `toml:"maxOpenConn"`
}

func (p *Postgres) HtmlReportName() string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("postgres_%s:%s_%s", p.Host, p.Port, p.DBName)
}

// MissingRequiredFields returns [postgres] keys that are empty (host, port, user, password, dbname).
func (p *Postgres) MissingRequiredFields() []string {
	if p == nil {
		return []string{"host", "port", "user", "password", "dbname"}
	}
	var missing []string
	for _, field := range []struct {
		value string
		key   string
	}{
		{strings.TrimSpace(p.Host), "host"},
		{strings.TrimSpace(p.Port), "port"},
		{strings.TrimSpace(p.User), "user"},
		{strings.TrimSpace(p.Password), "password"},
	} {
		if field.value == "" {
			missing = append(missing, field.key)
		}
	}
	if len(SplitDBNames(p.DBName)) == 0 {
		missing = append(missing, "dbname")
	}
	return missing
}

// Validate reports all missing required [postgres] fields before opening a connection.
func (p *Postgres) Validate() error {
	missing := p.MissingRequiredFields()
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf(
		"postgres config incomplete: missing [postgres] %s in kshieldconfig.toml (required: host, port, user, password, dbname)",
		strings.Join(missing, ", "),
	)
}

// SplitDBNames parses dbname from config. Supports a single name or comma-separated names (e.g. "hej, hej1").
func SplitDBNames(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// PrimaryDBName returns the first database name when dbname is comma-separated.
func (p *Postgres) PrimaryDBName() string {
	if p == nil {
		return ""
	}
	names := SplitDBNames(p.DBName)
	if len(names) == 0 {
		return strings.TrimSpace(p.DBName)
	}
	return names[0]
}

// ExpandTargets returns one Postgres config per database name (same host/port/user/password).
func (p *Postgres) ExpandTargets() []*Postgres {
	if p == nil {
		return nil
	}
	names := SplitDBNames(p.DBName)
	if len(names) == 0 {
		return nil
	}
	out := make([]*Postgres, 0, len(names))
	for _, db := range names {
		cp := *p
		cp.DBName = db
		out = append(out, &cp)
	}
	return out
}

// Open opens a the postgres database connection specified by its connection
// url which can be of format:
// https://pkg.go.dev/github.com/lib/pq#hdr-Connection_String_Parameters

var re = regexp.MustCompile(`(?m)(?:host=)([^\s]+)`)

// BuildConnectionString builds a PostgreSQL connection string from the given configuration
func BuildConnectionString(conf Postgres) string {
	var parts []string

	parts = append(parts,
		fmt.Sprintf("host=%s", conf.Host),
		fmt.Sprintf("port=%s", conf.Port),
		fmt.Sprintf("user=%s", conf.User),
		fmt.Sprintf("password=%s", conf.Password),
		fmt.Sprintf("dbname=%s", conf.DBName),
	)

	if conf.SSLmode != "" {
		parts = append(parts, fmt.Sprintf("sslmode=%s", conf.SSLmode))
	} else {
		parts = append(parts, "sslmode=disable")
	}
	if conf.SSLcert != "" {
		parts = append(parts, fmt.Sprintf("sslcert=%s", conf.SSLcert))
	}
	if conf.SSLkey != "" {
		parts = append(parts, fmt.Sprintf("sslkey=%s", conf.SSLkey))
	}
	if conf.SSLrootcert != "" {
		parts = append(parts, fmt.Sprintf("sslrootcert=%s", conf.SSLrootcert))
	}

	return strings.Join(parts, " ")
}

func Open(conf Postgres) (*sql.DB, string, error) {
	conf.DBName = conf.PrimaryDBName()
	if err := conf.Validate(); err != nil {
		log.Error().Err(err).Msg("postgres config validation failed")
		return nil, "", err
	}

	url := BuildConnectionString(conf)

	db, err := ConnectDatabaseUsingConnectionString(url, conf.PingCheck)
	if err != nil {
		return nil, "", err
	}
	if conf.MaxIdleConn > 0 {
		db.SetMaxIdleConns(conf.MaxIdleConn)
	}
	if conf.MaxOpenConn > 0 {
		db.SetMaxOpenConns(conf.MaxOpenConn)
	}

	// log.Info().
	// 	Int("Max open connections", conf.MaxOpenConn).
	// 	Int("Max idle connections", conf.MaxIdleConn).
	// 	Msg("Database connected successfully")
	// fmt.Println("Database connected successfully")
	// Extract hostname from connection string
	hostnameGroup := re.FindStringSubmatch(url)
	var hostname string
	if len(hostnameGroup) < 2 {
		log.Error().Msg("Failed to extract hostname from connection string")
		hostname = "unknown"
	} else {
		hostname = hostnameGroup[1]
	}

	return db, hostname, nil
}

// ConnectDatabaseUsingConnectionString connects to a PostgreSQL database using the provided connection string.
// It returns a database connection, the connection string, and an error if any.
func ConnectDatabaseUsingConnectionString(url string, pingCheck bool) (*sql.DB, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		log.Error().
			Err(err).
			Str("conn", url).
			Msg("Failed to open database connection")
		return nil, err
	}

	if pingCheck {
		err = db.Ping()
		if err != nil {
			log.Error().
				Err(err).
				Str("conn", url).
				Msg("Failed to ping database")
			db.Close()
			return nil, err
		}
	}

	return db, nil
}
