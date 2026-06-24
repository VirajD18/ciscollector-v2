package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

type usersReportSection struct {
	Title string
	Table usersReportTable
}

type usersReportTable struct {
	Columns []string
	Rows    [][]string
}

func decodeUsersReportSections(report map[string]interface{}) []usersReportSection {
	raw, ok := report["Users Report"]
	if !ok {
		return nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var sections []map[string]interface{}
	if err := json.Unmarshal(b, &sections); err != nil {
		return nil
	}
	var out []usersReportSection
	for _, sec := range sections {
		title, _ := sec["Title"].(string)
		data, _ := sec["Data"].(map[string]interface{})
		if data == nil {
			continue
		}
		tbl, _ := data["Table"].(map[string]interface{})
		if tbl == nil {
			continue
		}
		ut := usersReportTable{}
		if cols, ok := tbl["Columns"].([]interface{}); ok {
			for _, c := range cols {
				ut.Columns = append(ut.Columns, fmt.Sprint(c))
			}
		}
		if rows, ok := tbl["Rows"].([]interface{}); ok {
			for _, row := range rows {
				cells, ok := row.([]interface{})
				if !ok {
					continue
				}
				var line []string
				for _, c := range cells {
					line = append(line, fmt.Sprint(c))
				}
				if len(line) > 0 {
					ut.Rows = append(ut.Rows, line)
				}
			}
		}
		out = append(out, usersReportSection{Title: title, Table: ut})
	}
	return out
}

func usersSectionByTitle(sections []usersReportSection, substr string) *usersReportSection {
	sub := strings.ToLower(substr)
	for i := range sections {
		if strings.Contains(strings.ToLower(sections[i].Title), sub) {
			return &sections[i]
		}
	}
	return nil
}

func isSystemRole(name string) bool {
	return strings.HasPrefix(strings.ToLower(name), "pg_")
}

func cellBool(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "true" || s == "t" || s == "1"
}

// loginRolesFromUsersList returns human login-capable roles from "List of db users".
func loginRolesFromUsersList(sections []usersReportSection) []string {
	sec := usersSectionByTitle(sections, "list of db users")
	if sec == nil {
		return nil
	}
	loginIdx := -1
	for i, c := range sec.Table.Columns {
		if strings.EqualFold(c, "rolcanlogin") {
			loginIdx = i
			break
		}
	}
	var roles []string
	for _, row := range sec.Table.Rows {
		if len(row) == 0 {
			continue
		}
		name := row[0]
		if isSystemRole(name) {
			continue
		}
		if loginIdx >= 0 && loginIdx < len(row) && !cellBool(row[loginIdx]) {
			continue
		}
		roles = append(roles, name)
	}
	return roles
}
