package service

import (
	"context"
	"fmt"
)

// Policies returns policy templates and host assignments (v1: static catalog + SQLite hosts).
func (s *Service) Policies(ctx context.Context) (*PoliciesResponse, error) {
	checks := []PolicyCheck{
		{ID: "postgres_cis", Label: "CIS benchmarks", Cmd: "postgres_cis", Menu: "2"},
		{ID: "hba_scanner", Label: "HBA scanner", Cmd: "hba_scanner", Menu: "3"},
		{ID: "pii_scan", Label: "PII scan", Cmd: "datascan", Menu: "4"},
		{ID: "log_parser", Label: "Log parser (pg_log)", Cmd: "logparser", Menu: "-"},
		{ID: "ssl_audit", Label: "SSL audit", Cmd: "ssl_audit", Menu: "15"},
		{ID: "config_audit", Label: "Config audit", Cmd: "config_audit", Menu: "16"},
		{ID: "password_leak", Label: "Password leakage", Cmd: "password_leak", Menu: "10"},
	}
	allIDs := make([]string, len(checks))
	for i, c := range checks {
		allIDs[i] = c.ID
	}
	templates := []PolicyTemplate{
		{ID: "prod_strict", Name: "Prod strict", Desc: "All checks - critical production", Checks: allIDs},
		{ID: "dev_light", Name: "Dev light", Desc: "CIS + config audit only", Checks: []string{"postgres_cis", "config_audit", "hba_scanner"}},
		{ID: "cis_only", Name: "CIS-only", Desc: "Compliance baseline", Checks: []string{"postgres_cis"}},
	}
	defs := []PolicyDefinition{
		{ID: "prod_critical_policy", Name: "prod_critical_policy", Checks: allIDs},
		{ID: "prod_standard_policy", Name: "prod_standard_policy", Checks: templates[1].Checks},
	}
	groups := []PolicyGroup{
		{Name: "production", Hosts: []string{}},
		{Name: "staging", Hosts: []string{}},
	}
	hostMap := []PolicyHostMap{}
	runs, _ := s.latestRunsByTarget(ctx)
	checkTotal := len(allIDs)
	for _, run := range runs {
		host := hostLabel(run)
		groups[0].Hosts = append(groups[0].Hosts, host)
		pass := run.TotalPass
		if pass > checkTotal {
			pass = checkTotal
		}
		hostMap = append(hostMap, PolicyHostMap{
			Host: host, Policy: "prod_critical_policy",
			Checks: fmt.Sprintf("%d/%d", pass, checkTotal),
		})
	}
	return &PoliciesResponse{
		Checks: checks, Templates: templates, Groups: groups,
		HostMap: hostMap, Definitions: defs,
	}, nil
}
