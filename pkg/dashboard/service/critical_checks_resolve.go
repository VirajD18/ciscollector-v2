package service

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/VirajD18/ciscollector-v2/model"
)

var rePasswordEncryption = regexp.MustCompile(`(?i)password_encryption\s*:\s*([^\s,]+)`)

func cisSearchBlob(r model.Result) string {
	parts := []string{
		r.Control, r.Title, r.Description, r.FailReason, r.Procedure,
		manualCheckText(r),
	}
	return strings.ToLower(strings.Join(parts, " "))
}

func manualCheckText(r model.Result) string {
	if r.ManualCheckData == nil {
		return ""
	}
	if chk, ok := r.ManualCheckData.(model.ManualCheckTableDescriptionAndList); ok {
		text := chk.Description + " " + strings.Join(chk.List, " ")
		if chk.Table != nil {
			text += " " + chk.Text()
		}
		return text
	}
	b, err := json.Marshal(r.ManualCheckData)
	if err != nil {
		return ""
	}
	return string(b)
}

func findCISForCritical(id int, results []model.Result) *model.Result {
	if len(results) == 0 {
		return nil
	}
	switch id {
	case 1:
		return findCISFirst(results, "password_encryption", "password complexity")
	case 2:
		if r := findCISFirst(results, "rolpassword", "pg_shadow", "pg_authid"); r != nil {
			return r
		}
		return findCISFirst(results, "md5", "password")
	case 6:
		return findCISFirst(results, "prosecdef", "security definer", "function privileges")
	case 9:
		return findCISByGUC(results, "listen_addresses")
	case 11:
		return findCISByGUC(results, "log_connections")
	case 12:
		return findCISByGUC(results, "log_disconnections")
	case 13:
		return findCISByGUC(results, "log_statement")
	case 14:
		return findCISByGUC(results, "log_line_prefix")
	case 15:
		if r := findCISByGUC(results, "log_destination"); r != nil {
			return r
		}
		return findCISByGUC(results, "logging_collector")
	case 16:
		return findCISFirst(results, "pgaudit", "shared_preload")
	case 17:
		return findCISFirst(results, "pgaudit.log", "pgaudit")
	case 20:
		return findCISFirst(results, "pgpassword", "profile")
	case 21:
		return findCISFirst(results, "pgpassword", "environment")
	case 22:
		return findCISFirst(results, "psql_history", "command history")
	case 24:
		return findCISFirst(results, "has_database_privilege", "public", "connect")
	default:
		return nil
	}
}

func findCISFirst(results []model.Result, parts ...string) *model.Result {
	for i := range results {
		blob := cisSearchBlob(results[i])
		for _, p := range parts {
			if p != "" && strings.Contains(blob, strings.ToLower(p)) {
				return &results[i]
			}
		}
	}
	return nil
}

func findCISByGUC(results []model.Result, guc string) *model.Result {
	guc = strings.ToLower(strings.TrimSpace(guc))
	for i := range results {
		blob := cisSearchBlob(results[i])
		if strings.Contains(blob, guc) {
			return &results[i]
		}
	}
	return nil
}

func resolveCriticalFromCIS(id int, r model.Result) string {
	if s := strings.TrimSpace(r.Status); s != "" && !strings.EqualFold(s, "Manual") {
		if strings.EqualFold(s, "Pass") || strings.EqualFold(s, "Fail") {
			return normalizeCriticalCheckStatus(s)
		}
	}
	switch id {
	case 1:
		return resolveSCRAM(r)
	case 2:
		return resolveMD5Passwords(r)
	case 9:
		return resolveListenAddresses(r)
	case 11:
		return resolveGUCOn(r, "log_connections")
	case 12:
		return resolveGUCOn(r, "log_disconnections")
	case 13:
		return resolveLogStatement(r)
	case 14:
		return resolveLogLinePrefix(r)
	case 15:
		return resolveLogDestination(r)
	case 16:
		return resolvePGAuditInstalled(r)
	case 17:
		return resolvePGAuditLog(r)
	case 24:
		return resolvePublicDBTable(r)
	default:
		return ""
	}
}

func resolveSCRAM(r model.Result) string {
	if v := gucSettingValue(r, "password_encryption"); v != "" {
		if strings.EqualFold(v, "scram-sha-256") {
			return "Pass"
		}
		return "Fail"
	}
	if m := rePasswordEncryption.FindStringSubmatch(manualCheckText(r)); len(m) > 1 {
		if strings.EqualFold(strings.TrimSpace(m[1]), "scram-sha-256") {
			return "Pass"
		}
		return "Fail"
	}
	return ""
}

func resolveMD5Passwords(r model.Result) string {
	rows, ok := manualCheckRows(r)
	if !ok {
		return ""
	}
	if len(rows) == 0 {
		return "Pass"
	}
	return "Fail"
}

func resolveListenAddresses(r model.Result) string {
	for _, item := range manualCheckList(r) {
		if strings.TrimSpace(item) == "*" {
			return "Fail"
		}
	}
	if len(manualCheckList(r)) > 0 {
		return "Pass"
	}
	return ""
}

func resolveGUCOn(r model.Result, guc string) string {
	v := gucSettingValue(r, guc)
	if v == "" {
		if strings.Contains(strings.ToLower(r.FailReason), guc) {
			return "Fail"
		}
		return ""
	}
	if strings.EqualFold(v, "on") {
		return "Pass"
	}
	return "Fail"
}

func resolveLogStatement(r model.Result) string {
	v := gucSettingValue(r, "log_statement")
	if v == "" {
		if strings.Contains(strings.ToLower(r.FailReason), "log_statement") {
			return "Fail"
		}
		return ""
	}
	if strings.EqualFold(v, "none") {
		return "Fail"
	}
	return "Pass"
}

func resolveLogLinePrefix(r model.Result) string {
	v := gucSettingValue(r, "log_line_prefix")
	if v == "" {
		if strings.Contains(strings.ToLower(r.FailReason), "log_line_prefix") {
			return "Fail"
		}
		return ""
	}
	required := []string{"%m", "%p", "%l", "%d", "%u", "%a", "%h"}
	for _, tok := range required {
		if !strings.Contains(v, tok) {
			return "Fail"
		}
	}
	return "Pass"
}

func resolveLogDestination(r model.Result) string {
	dest := gucSettingValue(r, "log_destination")
	collector := gucSettingValue(r, "logging_collector")
	if dest == "" && collector == "" {
		if strings.Contains(strings.ToLower(r.FailReason), "log_destination") ||
			strings.Contains(strings.ToLower(r.FailReason), "logging_collector") {
			return "Fail"
		}
		return ""
	}
	if strings.EqualFold(collector, "on") && (strings.Contains(strings.ToLower(dest), "csvlog") || dest != "") {
		return "Pass"
	}
	if strings.Contains(strings.ToLower(dest), "syslog") {
		return "Pass"
	}
	if dest != "" && !strings.EqualFold(dest, "stderr") {
		return "Pass"
	}
	if strings.EqualFold(collector, "off") && strings.EqualFold(dest, "stderr") {
		return "Fail"
	}
	if dest != "" {
		return "Pass"
	}
	return "Fail"
}

func resolvePGAuditInstalled(r model.Result) string {
	v := gucSettingValue(r, "shared_preload_libraries")
	if v == "" {
		if strings.Contains(strings.ToLower(r.FailReason), "pgaudit") {
			return "Fail"
		}
		return ""
	}
	if strings.Contains(strings.ToLower(v), "pgaudit") {
		return "Pass"
	}
	return "Fail"
}

func resolvePGAuditLog(r model.Result) string {
	v := gucSettingValue(r, "pgaudit.log")
	if v == "" {
		if strings.Contains(strings.ToLower(r.FailReason), "pgaudit") {
			return "Fail"
		}
		return ""
	}
	lower := strings.ToLower(v)
	for _, part := range []string{"role", "ddl", "write"} {
		if !strings.Contains(lower, part) {
			return "Fail"
		}
	}
	return "Pass"
}

func resolvePublicDBTable(r model.Result) string {
	rows, ok := manualCheckRows(r)
	if !ok {
		return ""
	}
	if len(rows) == 0 {
		return "Pass"
	}
	return "Fail"
}

func gucSettingValue(r model.Result, guc string) string {
	for name, val := range gucValuesFromManualCheck(r) {
		if strings.EqualFold(name, guc) {
			return val
		}
	}
	return ""
}

func manualCheckList(r model.Result) []string {
	if r.ManualCheckData == nil {
		return nil
	}
	if chk, ok := r.ManualCheckData.(model.ManualCheckTableDescriptionAndList); ok {
		return chk.List
	}
	return nil
}

func manualCheckRows(r model.Result) ([][]interface{}, bool) {
	if r.ManualCheckData == nil {
		return nil, false
	}
	chk, ok := r.ManualCheckData.(model.ManualCheckTableDescriptionAndList)
	if !ok {
		return nil, false
	}
	if chk.Table == nil {
		return nil, false
	}
	return chk.Table.Rows, true
}
