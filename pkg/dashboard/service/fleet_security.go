package service

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/klouddb/klouddbshield/pkg/reportstore"
)

func (agg *fleetAccumulator) collectHBAIssues(host string, report map[string]interface{}) {
	for _, h := range decodeHBAResults(report) {
		if !strings.EqualFold(h.Status, "Fail") {
			continue
		}
		agg.hbaHosts[host] = true
		check := strings.TrimSpace(h.Title)
		if check == "" {
			check = fmt.Sprintf("HBA check %d", h.Control)
		}
		agg.hbaRows = append(agg.hbaRows, []string{host, check, h.Status, "Open"})
	}
}

func (agg *fleetAccumulator) collectUsageOfDefaults(host string, run *reportstore.RunRow, sections []usersReportSection) {
	if issue, detail := hostUsesDefaults(run, sections); issue {
		agg.defaultsHosts[host] = true
		agg.defaultsRows = append(agg.defaultsRows, []string{host, "Default port + postgres role", detail, "Open"})
	}
}

func hostUsesDefaults(run *reportstore.RunRow, sections []usersReportSection) (bool, string) {
	port := "5432"
	if run != nil && strings.TrimSpace(run.TargetPort) != "" {
		port = strings.TrimSpace(run.TargetPort)
	}
	hasPostgres := false
	for _, role := range loginRolesFromUsersList(sections) {
		if strings.EqualFold(role, "postgres") {
			hasPostgres = true
			break
		}
	}
	if port == "5432" && hasPostgres {
		return true, "Default port 5432 with login role postgres"
	}
	return false, ""
}

func (agg *fleetAccumulator) collectSuperuserCounts(host string, sections []usersReportSection) {
	count, names, overLimit := superuserCountIssue(sections)
	if !overLimit {
		return
	}
	agg.superuserHosts[host] = true
	roleSummary := strings.Join(names, ", ")
	if roleSummary == "" {
		roleSummary = "-"
	}
	agg.superuserRows = append(agg.superuserRows, []string{
		host, fmt.Sprintf("%d", count), roleSummary, "Open",
	})
}

func superuserCountIssue(sections []usersReportSection) (count int, names []string, overLimit bool) {
	if sec := usersSectionByTitle(sections, "superuser count"); sec != nil && len(sec.Table.Rows) > 0 {
		row := sec.Table.Rows[0]
		if countStr := columnValueByName(row, sec.Table.Columns, "superuser_count"); countStr != "" {
			if n, err := strconv.Atoi(countStr); err == nil {
				count = n
			}
			passIdx, _ := passColumnIndices(sec.Table.Columns)
			if passIdx >= 0 {
				return count, nil, !cellBool(columnValue(row, passIdx))
			}
		}
	}

	sec := usersSectionByTitle(sections, "superuser")
	if sec == nil {
		return 0, nil, false
	}
	for _, row := range sec.Table.Rows {
		if len(row) == 0 || isSystemRole(row[0]) {
			continue
		}
		count++
		names = append(names, row[0])
	}
	return count, names, count > 3
}
