package config

import (
	"strings"

	"github.com/klouddb/klouddbshield/pkg/postgresdb"
)

// ResolvePostgresTarget returns connection settings for a PII scan target.
// Matches [[crons.commands.postgres]] by port when set; otherwise uses [postgres].
func (c *Config) ResolvePostgresTarget(targetHost, targetPort, database string) *postgresdb.Postgres {
	var base *postgresdb.Postgres
	if fp := c.FirstPostgres(); fp != nil {
		cp := *fp
		base = &cp
	} else {
		base = &postgresdb.Postgres{Host: "localhost", Port: "5432", User: "postgres", DBName: "postgres"}
	}

	wantPort := strings.TrimSpace(targetPort)
	if wantPort != "" {
		if found := c.findCronPostgresByPort(wantPort); found != nil {
			cp := *found
			base = &cp
		}
	}

	if h := strings.TrimSpace(targetHost); h != "" {
		base.Host = h
	}
	if wantPort != "" {
		base.Port = wantPort
	}
	if db := strings.TrimSpace(database); db != "" {
		base.DBName = db
	}
	return base
}

func (c *Config) findCronPostgresByPort(port string) *postgresdb.Postgres {
	for _, cron := range c.Crons {
		for _, cmd := range cron.Commands {
			for _, pg := range cmd.Postgres {
				if pg != nil && pg.Port == port {
					return pg
				}
			}
		}
	}
	return nil
}
