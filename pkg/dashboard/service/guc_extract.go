package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/klouddb/klouddbshield/model"
)

var (
	reQuotedGUC       = regexp.MustCompile(`'([^']+)'`)
	reShowSetting     = regexp.MustCompile(`(?i)show\s+([a-z][a-z0-9_]*)\s*;`)
	reFailSetting     = regexp.MustCompile(`^([a-z][a-z0-9_]*)\s+is\s+not\b`)
	reSnakeGucName    = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	reCamelGucName    = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)
	gucTitleBlocklist = map[string]bool{
		"postmaster": true, "sighup": true, "superuser": true, "user": true,
	}
)

func gucValuesFromManualCheck(r model.Result) map[string]string {
	if r.ManualCheckData == nil {
		return nil
	}
	b, err := json.Marshal(r.ManualCheckData)
	if err != nil {
		return nil
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil
	}
	tbl, _ := raw["Table"].(map[string]interface{})
	if tbl == nil {
		return nil
	}
	cols, _ := tbl["Columns"].([]interface{})
	rows, _ := tbl["Rows"].([]interface{})
	if len(cols) < 2 || len(rows) == 0 {
		return nil
	}
	nameIdx, settingIdx := -1, -1
	for i, c := range cols {
		cl := strings.ToLower(strings.TrimSpace(toString(c)))
		switch cl {
		case "name":
			nameIdx = i
		case "setting":
			settingIdx = i
		}
	}
	// Only pg_settings query tables (name + setting); skip pg_hba_file_rules, roles, etc.
	if nameIdx < 0 || settingIdx < 0 {
		return nil
	}
	out := map[string]string{}
	for _, row := range rows {
		cells, ok := row.([]interface{})
		if !ok || len(cells) <= settingIdx || len(cells) <= nameIdx {
			continue
		}
		name := strings.TrimSpace(toString(cells[nameIdx]))
		if !looksLikePostgresGucName(name) {
			continue
		}
		out[name] = strings.TrimSpace(toString(cells[settingIdx]))
	}
	return out
}

func extractGucName(r model.Result) string {
	if n := quotedGucFromText(r.Title); n != "" {
		return n
	}
	if n := showSettingFromText(r.Procedure); n != "" {
		return n
	}
	if n := failSettingFromText(r.FailReason); n != "" {
		return n
	}
	return ""
}

func quotedGucFromText(s string) string {
	m := reQuotedGUC.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	name := strings.TrimSpace(m[1])
	if strings.Contains(name, " ") || !looksLikePostgresGucName(name) {
		return ""
	}
	return name
}

func showSettingFromText(s string) string {
	m := reShowSetting.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func failSettingFromText(s string) string {
	m := reFailSetting.FindStringSubmatch(strings.TrimSpace(s))
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func looksLikeControlID(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if !strings.Contains(s, ".") {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && r != '.' {
			return false
		}
	}
	return true
}

func looksLikePostgresGucName(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > 64 {
		return false
	}
	if looksLikeControlID(s) {
		return false
	}
	if strings.ContainsAny(s, "/\\ .") {
		return false
	}
	if matched, _ := regexp.MatchString(`^\d+$`, s); matched {
		return false
	}
	if gucTitleBlocklist[strings.ToLower(s)] {
		return false
	}
	return reSnakeGucName.MatchString(s) || reCamelGucName.MatchString(s)
}

func gucExpectedFromCIS(r model.Result) string {
	title := strings.ToLower(r.Title)
	switch {
	case strings.Contains(title, "enabled"):
		return "on"
	case strings.Contains(title, "disabled"):
		return "off"
	case strings.Contains(title, "set correctly"):
		return "per CIS profile"
	}
	if strings.TrimSpace(r.Description) != "" {
		return trimForTable(r.Description, 48)
	}
	return "CIS expected"
}

func gucDriftLiveValue(r model.Result) string {
	if live := strings.TrimSpace(violationDetailsFromCIS(r)); live != "" {
		if !isNoisyFailReason(live) && len(live) <= 48 {
			return live
		}
	}
	return gucLiveValue(r)
}

func gucLiveValue(r model.Result) string {
	title := strings.ToLower(r.Title)
	if strings.EqualFold(r.Status, "Pass") {
		switch {
		case strings.Contains(title, "disabled"):
			return "off"
		case strings.Contains(title, "enabled"):
			return "on"
		default:
			return "ok"
		}
	}
	if fr := strings.TrimSpace(r.FailReason); fr != "" {
		if len(fr) <= 40 {
			return fr
		}
		return trimForTable(fr, 40)
	}
	switch {
	case strings.Contains(title, "enabled"):
		return "off"
	case strings.Contains(title, "disabled"):
		return "on"
	}
	return "-"
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}
