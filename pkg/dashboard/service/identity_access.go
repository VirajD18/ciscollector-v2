package service

import (
	"strings"

	"github.com/klouddb/klouddbshield/pkg/reportstore"
)

// buildIdentityAccess derives Identity & access widgets from Users Report in SQLite scans.
func buildIdentityAccess(runs []*reportstore.RunRow, passwordLeakHosts map[string]bool) ([]StrategicBar, StrategicHygiene, StrategicCred) {
	var privs []StrategicBar
	var activeRoles, inactiveRoles, commonRoles int
	weakCredHosts := map[string]bool{}

	for _, run := range runs {
		host := hostLabel(run)
		sections := decodeUsersReportSections(run.Report)
		loginRoles := loginRolesFromUsersList(sections)
		elevated := elevatedLoginRolesSet(sections)
		expiryUnset := humanRoleNamesInSection(sections, "password expiry")
		noConnLimit := humanRoleNamesInSection(sections, "without connection limits")

		adminPct := 0
		if len(loginRoles) > 0 {
			elevN := 0
			for _, role := range loginRoles {
				if elevated[role] {
					elevN++
				}
			}
			adminPct = elevN * 100 / len(loginRoles)
		}
		privs = append(privs, StrategicBar{
			L: hostDisplayLabel(host),
			A: adminPct,
			U: 100 - adminPct,
		})

		for _, role := range loginRoles {
			switch {
			case expiryUnset[role]:
				inactiveRoles++
				weakCredHosts[host] = true
			case elevated[role]:
				commonRoles++
			default:
				activeRoles++
			}
		}
		if len(loginRoles) == 0 {
			for role := range expiryUnset {
				if !isSystemRole(role) {
					weakCredHosts[host] = true
				}
			}
		}
		for role := range noConnLimit {
			if !isSystemRole(role) {
				weakCredHosts[host] = true
			}
		}
	}

	hygiene := StrategicHygiene{}
	roleTotal := activeRoles + inactiveRoles + commonRoles
	if roleTotal > 0 {
		hygiene.Active = activeRoles * 100 / roleTotal
		hygiene.Inactive = inactiveRoles * 100 / roleTotal
		hygiene.Common = commonRoles * 100 / roleTotal
	} else if len(runs) > 0 {
		// Users Report missing or no login roles — neutral shell (not CIS host status).
		hygiene.Active = 100
	}

	exposed := len(passwordLeakHosts)
	weak := len(weakCredHosts)
	hostTotal := len(runs)
	ok := hostTotal - exposed - weak
	if ok < 0 {
		ok = 0
	}
	if exposed == 0 && weak == 0 && ok == 0 && hostTotal > 0 {
		ok = hostTotal
	}

	cred := StrategicCred{
		Hosts:   exposed,
		Exposed: exposed,
		Weak:    weak,
		Ok:      ok,
	}
	return privs, hygiene, cred
}

func elevatedLoginRolesSet(sections []usersReportSection) map[string]bool {
	out := map[string]bool{}
	for _, sub := range []string{"superuser", "createdb", "createrole", "bypassrls"} {
		for name := range humanRoleNamesInSection(sections, sub) {
			out[name] = true
		}
	}
	return out
}

func humanRoleNamesInSection(sections []usersReportSection, titleSubstr string) map[string]bool {
	sec := usersSectionByTitle(sections, titleSubstr)
	out := map[string]bool{}
	if sec == nil {
		return out
	}
	for _, row := range sec.Table.Rows {
		if len(row) == 0 || isSystemRole(row[0]) {
			continue
		}
		out[row[0]] = true
	}
	return out
}

func hostDisplayLabel(host string) string {
	h := host
	if i := strings.Index(h, ":"); i > 0 {
		h = h[:i]
	}
	switch strings.ToLower(h) {
	case "localhost":
		return "Local"
	default:
		if len(h) > 14 {
			return h[:14]
		}
		return h
	}
}
