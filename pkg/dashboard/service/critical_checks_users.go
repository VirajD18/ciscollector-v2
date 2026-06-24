package service

import (
	"fmt"
	"strings"
)

// usersPassSectionSubstr maps Top 25 check IDs to Users Report section title substrings.
var usersPassSectionSubstr = map[int]string{
	1:  "scram-sha-256 password_encryption",
	2:  "non-scram password hashes",
	6:  "security definer functions",
	7:  "superuser count",
	9:  "listen_addresses not wildcard",
	11: "log_connections enabled",
	12: "log_disconnections enabled",
	13: "log_statement at least ddl",
	14: "log_line_prefix format",
	15: "log_destination persistent",
	16: "pgaudit extension installed",
	17: "pgaudit.log configured",
}

var usersListSectionSubstr = map[int]string{
	24: "open to public connect",
}

func evalFromUsersReport(ctx criticalEvalContext, id int, title string) (CriticalCheckResult, bool) {
	if sub, ok := usersListSectionSubstr[id]; ok {
		sec := usersSectionByTitle(ctx.sections, sub)
		if sec == nil {
			return CriticalCheckResult{}, false
		}
		return evalUsersReportListSection(id, title, sec), true
	}
	sub, ok := usersPassSectionSubstr[id]
	if !ok {
		return CriticalCheckResult{}, false
	}
	sec := usersSectionByTitle(ctx.sections, sub)
	if sec == nil {
		return CriticalCheckResult{}, false
	}
	return evalUsersReportPassSection(id, title, sec), true
}

func evalUsersReportListSection(id int, title string, sec *usersReportSection) CriticalCheckResult {
	base := CriticalCheckResult{ID: id, Title: title, Source: "Users Report"}
	if len(sec.Table.Rows) == 0 {
		base.Status = "Pass"
		base.Details = usersPassDetails(id, "", true)
		return base
	}
	names := make([]string, 0, len(sec.Table.Rows))
	for _, row := range sec.Table.Rows {
		if len(row) > 0 {
			names = append(names, row[0])
		}
	}
	base.Status = "Fail"
	base.Details = "Databases open to PUBLIC: " + strings.Join(names, ", ")
	return base
}

func preferUsersThenCIS(ctx criticalEvalContext, id int, title string, parts ...string) CriticalCheckResult {
	if r, ok := evalFromUsersReport(ctx, id, title); ok {
		return r
	}
	return evalCriticalCIS(ctx, id, title, parts...)
}

func evalUsersReportPassSection(id int, title string, sec *usersReportSection) CriticalCheckResult {
	base := CriticalCheckResult{ID: id, Title: title, Source: "Users Report"}
	if len(sec.Table.Rows) == 0 {
		base.Status = "Manual"
		return base
	}
	passIdx, settingIdx := passColumnIndices(sec.Table.Columns)
	row := sec.Table.Rows[0]
	passVal := columnValue(row, passIdx)
	settingVal := columnValue(row, settingIdx)

	if passIdx < 0 {
		base.Status = "Manual"
		base.Details = "Users Report missing pass column"
		return base
	}
	if cellBool(passVal) {
		base.Status = "Pass"
		base.Details = usersPassDetails(id, settingVal, true)
		return base
	}
	base.Status = "Fail"
	base.Details = usersPassDetails(id, settingVal, false)
	switch id {
	case 7:
		if countVal := columnValueByName(row, sec.Table.Columns, "superuser_count"); countVal != "" {
			base.Details = fmt.Sprintf("%s superuser role(s) — limit is 3", countVal)
		}
	case 2:
		if countVal := columnValueByName(row, sec.Table.Columns, "non_scram_count"); countVal != "" && countVal != "0" {
			base.Details = countVal + " role(s) with non-SCRAM password hashes"
		}
	case 6:
		if countVal := columnValueByName(row, sec.Table.Columns, "secdef_count"); countVal != "" && countVal != "0" {
			base.Details = countVal + " SECURITY DEFINER function(s) found"
		}
	}
	return base
}

func passColumnIndices(cols []string) (passIdx, settingIdx int) {
	passIdx, settingIdx = -1, -1
	for i, c := range cols {
		cl := strings.ToLower(strings.TrimSpace(c))
		switch cl {
		case "pass":
			passIdx = i
		case "setting", "password_encryption", "listen_addresses", "log_line_prefix",
			"pgaudit_log", "shared_preload_libraries", "log_destination", "logging_collector",
			"superuser_count", "non_scram_count", "secdef_count", "public_db_count":
			settingIdx = i
		}
	}
	return passIdx, settingIdx
}

func columnValue(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func columnValueByName(row []string, cols []string, name string) string {
	name = strings.ToLower(name)
	for i, c := range cols {
		if strings.EqualFold(strings.TrimSpace(c), name) && i < len(row) {
			return strings.TrimSpace(row[i])
		}
	}
	return ""
}

func usersPassDetails(id int, setting string, pass bool) string {
	if setting != "" {
		if pass {
			return fmt.Sprintf("check passed — %s", setting)
		}
		return fmt.Sprintf("check failed — %s", setting)
	}
	switch id {
	case 1:
		if pass {
			return "password_encryption = scram-sha-256"
		}
		return "password_encryption is not scram-sha-256"
	case 2:
		if pass {
			return "All role passwords use SCRAM-SHA-256"
		}
		return "Non-SCRAM password hashes found"
	case 6:
		if pass {
			return "No SECURITY DEFINER functions in user schemas"
		}
		return "SECURITY DEFINER functions require review"
	case 7:
		if pass {
			return "Superuser count within limit of 3"
		}
		return "More than 3 superuser roles"
	case 9:
		if pass {
			return "listen_addresses is not set to *"
		}
		return "listen_addresses is set to *"
	case 11:
		if pass {
			return "log_connections = on"
		}
		return "log_connections is not on"
	case 12:
		if pass {
			return "log_disconnections = on"
		}
		return "log_disconnections is not on"
	case 13:
		if pass {
			return "log_statement is ddl, mod, or all"
		}
		return "log_statement is none"
	case 14:
		if pass {
			return "log_line_prefix includes required tokens"
		}
		return "log_line_prefix missing required format"
	case 15:
		if pass {
			return "log_destination writes to persistent location"
		}
		return "log_destination not persistent (stderr only)"
	case 16:
		if pass {
			return "shared_preload_libraries includes pgaudit"
		}
		return "pgaudit not in shared_preload_libraries"
	case 17:
		if pass {
			return "pgaudit.log includes role, ddl, write"
		}
		return "pgaudit.log missing role, ddl, or write"
	case 24:
		if pass {
			return "No databases grant CONNECT to PUBLIC"
		}
		return "Databases grant CONNECT to PUBLIC"
	default:
		if pass {
			return "Pass"
		}
		return "Fail"
	}
}
