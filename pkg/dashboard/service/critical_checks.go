package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/klouddb/klouddbshield/model"
	"github.com/klouddb/klouddbshield/pkg/reportstore"
	"github.com/klouddb/klouddbshield/pkg/repository"
)

var criticalCheckTitles = []string{
	"SCRAM-SHA-256 Enforced",
	"MD5 Authentication Disabled",
	"PII Data Exposed",
	"Leaked Passwords Detected",
	"SSL Violations Detected",
	"SECURITY DEFINER Functions Reviewed",
	"Superuser Count Minimized",
	"No Trust Authentication In pg_hba.conf",
	"listen_addresses Is Not Set To *",
	"pg_hba.conf Does Not Contain 0.0.0.0/0 Or ::/0 Entries",
	"log_connections = On",
	"log_disconnections = On",
	"log_statement At Least DDL",
	"log_line_prefix Includes Required Format",
	"log_destination Writes To A Persistent Location",
	"pgaudit Extension Installed And Configured",
	"pgaudit.log Includes Role, DDL, Write At Minimum",
	"Common Usernames Detected",
	"pg_hba.conf Does Not Contain hostssl",
	"Verify That 'PGPASSWORD' Is Not Set In Users' Profiles",
	"Verify That The 'PGPASSWORD' Environment Variable Is Not In Use",
	"Disable PostgreSQL Command History",
	"Ensure Per-Account Connection Limits Are Used",
	"Databases Open To Public",
	"Usage Of Default Port 5432 And Default Admin Role postgres",
}

var commonDBUsernames = map[string]bool{
	"postgres": true, "admin": true, "root": true, "test": true,
	"guest": true, "user": true, "oracle": true, "mysql": true,
}

type criticalEvalContext struct {
	cis      []model.Result
	hba      []model.HBAScannerResult
	sections []usersReportSection
	report   map[string]interface{}
	pii      map[string]interface{}
	run      *reportstore.RunRow
}

// CriticalChecksForRun evaluates canonical critical checks for one scan run.
// PII check #3 uses run.PiiReport only; prefer criticalChecksForRun on Service for PII fallback.
func CriticalChecksForRun(run *reportstore.RunRow) []CriticalCheckResult {
	return criticalChecksForRun(run, nil)
}

func criticalChecksForRun(run *reportstore.RunRow, piiReport map[string]interface{}) []CriticalCheckResult {
	if run == nil {
		return nil
	}
	pii := piiReport
	if len(pii) == 0 {
		pii = run.PiiReport
	}
	ctx := criticalEvalContext{
		cis:      decodeCISResults(run.Report),
		hba:      decodeHBAResults(run.Report),
		sections: decodeUsersReportSections(run.Report),
		report:   run.Report,
		pii:      pii,
		run:      run,
	}
	out := make([]CriticalCheckResult, 0, len(criticalCheckTitles))
	for i, title := range criticalCheckTitles {
		out = append(out, evalCriticalCheck(ctx, i+1, title))
	}
	return out
}

// resolvePIIReport returns PII data for check #3: current run first, else newest pii_report_json for target.
func resolvePIIReport(ctx context.Context, repo repository.Repository, run *reportstore.RunRow) map[string]interface{} {
	if run == nil {
		return nil
	}
	if len(run.PiiReport) > 0 {
		return run.PiiReport
	}
	if repo == nil || run.TargetID == "" {
		return nil
	}
	withPII, err := repo.GetLatestRunWithPII(ctx, run.TargetID)
	if err != nil || withPII == nil {
		return nil
	}
	return withPII.PiiReport
}

func (s *Service) criticalChecksForRun(ctx context.Context, run *reportstore.RunRow) []CriticalCheckResult {
	return criticalChecksForRun(run, resolvePIIReport(ctx, s.Repo, run))
}

// CriticalChecksFleet returns fleet-wide critical check results from latest SQLite scans.
func (s *Service) CriticalChecksFleet(ctx context.Context) (*CriticalChecksResponse, error) {
	runs, err := s.latestRunsByTarget(ctx)
	if err != nil {
		return nil, err
	}
	resp := &CriticalChecksResponse{
		Checks:     criticalCheckDefinitions(),
		HostRows:   []CriticalCheckHostRow{},
		CheckFails: make([]int, len(criticalCheckTitles)),
	}
	for _, run := range runs {
		host := hostLabel(run)
		detected := formatDetectedAt(run.StartedAt)
		checks := s.criticalChecksForRun(ctx, run)
		failed := 0
		for _, c := range checks {
			if strings.EqualFold(c.Status, "Fail") {
				failed++
				if c.ID >= 1 && c.ID <= len(resp.CheckFails) {
					resp.CheckFails[c.ID-1]++
				}
			}
		}
		resp.HostRows = append(resp.HostRows, CriticalCheckHostRow{
			Host:     host,
			Detected: detected,
			Failed:   failed,
			Checks:   checks,
		})
		for _, c := range checks {
			if !strings.EqualFold(c.Status, "Fail") {
				continue
			}
			vtype := violationTypeForCriticalCheck(c.ID, c.Title, c.Source)
			sev := severityForCriticalCheck(c.ID)
			resp.Rows = append(resp.Rows, CriticalCheckRow{
				ID:            fmt.Sprintf("V-%03d-%s", c.ID, slugHost(host)),
				CheckID:       c.ID,
				Check:         c.Title,
				Server:        host,
				Details:       c.Details,
				Status:        "Open",
				Detected:      detected,
				Source:        c.Source,
				ViolationType: vtype,
				Severity:      sev,
			})
		}
	}
	resp.CheckOptions = criticalCheckDefinitionFilterOptions(resp.Checks)
	resp.ServerOptions = criticalCheckServerFilterOptions(resp.Rows)
	resp.SourceOptions = criticalCheckAllSourceFilterOptions(resp.Rows)
	resp.TypeOptions = criticalCheckAllTypeFilterOptions()
	resp.SeverityOptions = criticalCheckAllSeverityFilterOptions()
	return resp, nil
}

func violationTypeForCriticalCheck(id int, title, source string) string {
	blob := strings.ToLower(title + " " + source)
	switch {
	case id == 3 || strings.Contains(blob, "pii"):
		return "PII Exposure"
	case id == 4 || strings.Contains(blob, "password") || strings.Contains(blob, "leak"):
		return "Password Leak"
	case id == 5 || id == 19 || strings.Contains(blob, "ssl") || strings.Contains(blob, "hostssl"):
		return "SSL Violation"
	case id == 7 || id == 18 || id == 23 || strings.Contains(blob, "superuser") || strings.Contains(blob, "username"):
		return "Unauthorized Superuser"
	case strings.Contains(strings.ToLower(source), "hba"):
		return "HBA Violation"
	default:
		return "Critical Config"
	}
}

func severityForCriticalCheck(id int) string {
	switch id {
	case 6, 15, 22, 24:
		return "HIGH"
	default:
		return "CRITICAL"
	}
}

func criticalCheckAllSourceFilterOptions(rows []CriticalCheckRow) []string {
	seen := map[string]bool{}
	var opts []string
	for _, src := range []string{"CIS", "HBA Scanner", "Users Report", "PII Scanner", "Log Parser", "Password Audit", "Host Inventory"} {
		seen[src] = true
		opts = append(opts, src)
	}
	for _, r := range rows {
		if r.Source != "" && !seen[r.Source] {
			seen[r.Source] = true
			opts = append(opts, r.Source)
		}
	}
	return opts
}

func criticalCheckAllTypeFilterOptions() []string {
	return []string{
		"SSL Violation",
		"HBA Violation",
		"Password Leak",
		"PII Exposure",
		"Unauthorized Superuser",
		"Critical Config",
	}
}

func criticalCheckAllSeverityFilterOptions() []string {
	return []string{"CRITICAL", "HIGH"}
}

func criticalCheckDefinitions() []CriticalCheckDef {
	out := make([]CriticalCheckDef, len(criticalCheckTitles))
	for i, title := range criticalCheckTitles {
		out[i] = CriticalCheckDef{ID: i + 1, Title: title}
	}
	return out
}

func criticalCheckDefinitionFilterOptions(defs []CriticalCheckDef) []string {
	opts := make([]string, 0, len(defs))
	for _, d := range defs {
		opts = append(opts, fmt.Sprintf("%d — %s", d.ID, d.Title))
	}
	return opts
}

func criticalCheckServerFilterOptions(rows []CriticalCheckRow) []string {
	seen := map[string]bool{}
	var opts []string
	for _, r := range rows {
		if r.Server == "" || seen[r.Server] {
			continue
		}
		seen[r.Server] = true
		opts = append(opts, r.Server)
	}
	return opts
}

func slugHost(host string) string {
	h := strings.NewReplacer(":", "-", ".", "-", "/", "-").Replace(host)
	if len(h) > 24 {
		return h[:24]
	}
	return h
}

func evalCriticalCheck(ctx criticalEvalContext, id int, title string) CriticalCheckResult {
	switch id {
	case 1:
		return preferUsersThenCIS(ctx, id, title, "password_encryption", "scram")
	case 2:
		return preferUsersThenCIS(ctx, id, title, "md5", "rolpassword")
	case 3:
		return evalCriticalPII(ctx, id, title)
	case 4:
		return evalCriticalPasswordLeak(ctx, id, title)
	case 5:
		return evalCriticalSSL(ctx, id, title)
	case 6:
		return preferUsersThenCIS(ctx, id, title, "security definer", "prosecdef", "function privileges")
	case 7:
		if r, ok := evalFromUsersReport(ctx, id, title); ok {
			return r
		}
		return evalCriticalSuperuserCount(ctx, id, title)
	case 8:
		return evalCriticalHBA(ctx, id, title, 1)
	case 9:
		return preferUsersThenCIS(ctx, id, title, "listen_addresses")
	case 10:
		return evalCriticalHBA(ctx, id, title, 9)
	case 11:
		return preferUsersThenCIS(ctx, id, title, "log_connections")
	case 12:
		return preferUsersThenCIS(ctx, id, title, "log_disconnections")
	case 13:
		return preferUsersThenCIS(ctx, id, title, "log_statement")
	case 14:
		return preferUsersThenCIS(ctx, id, title, "log_line_prefix")
	case 15:
		return preferUsersThenCIS(ctx, id, title, "log_destination", "logging_collector")
	case 16:
		return preferUsersThenCIS(ctx, id, title, "pgaudit", "shared_preload")
	case 17:
		return preferUsersThenCIS(ctx, id, title, "pgaudit.log")
	case 18:
		return evalCriticalCommonUsers(ctx, id, title)
	case 19:
		return evalCriticalHBA(ctx, id, title, 8)
	case 20:
		return evalCriticalCIS(ctx, id, title, "pgpassword", "profile")
	case 21:
		return evalCriticalCIS(ctx, id, title, "pgpassword", "environment")
	case 22:
		return evalCriticalCIS(ctx, id, title, "psql_history", "command history")
	case 23:
		return evalCriticalConnLimits(ctx, id, title)
	case 24:
		if r, ok := evalFromUsersReport(ctx, id, title); ok {
			return r
		}
		return evalCriticalCIS(ctx, id, title, "public", "connect", "has_database_privilege")
	case 25:
		return evalCriticalDefaultPortPostgres(ctx, id, title)
	default:
		return CriticalCheckResult{ID: id, Title: title, Status: "Manual"}
	}
}

func evalCriticalCIS(ctx criticalEvalContext, id int, title string, parts ...string) CriticalCheckResult {
	r := findCISForCritical(id, ctx.cis)
	if r == nil {
		r = findCISMatch(ctx.cis, parts...)
	}
	if r == nil {
		return evalCriticalCISFallback(ctx, id, title)
	}
	return checkFromCIS(*r, id, title)
}

func evalCriticalCISFallback(ctx criticalEvalContext, id int, title string) CriticalCheckResult {
	return CriticalCheckResult{ID: id, Title: title, Status: "Manual", Source: "CIS"}
}

func evalCriticalHBA(ctx criticalEvalContext, id int, title string, control int) CriticalCheckResult {
	h := findHBAMatch(ctx.hba, control)
	if h == nil {
		return CriticalCheckResult{ID: id, Title: title, Status: "Manual", Source: "HBA Scanner"}
	}
	return checkFromHBA(*h, id, title)
}

func checkFromCIS(r model.Result, id int, title string) CriticalCheckResult {
	status := normalizeCriticalCheckStatus(r.Status)
	if strings.EqualFold(status, "Manual") {
		if resolved := resolveCriticalFromCIS(id, r); resolved != "" {
			status = resolved
		}
	}
	details := violationDetailsFromCIS(r)
	if details == "" {
		details = strings.TrimSpace(r.Title)
	}
	return CriticalCheckResult{
		ID:      id,
		Title:   title,
		Status:  status,
		Details: details,
		Source:  "CIS",
	}
}

func checkFromHBA(h model.HBAScannerResult, id int, title string) CriticalCheckResult {
	details := strings.TrimSpace(h.Description)
	if details == "" {
		details = strings.TrimSpace(h.Title)
	}
	return CriticalCheckResult{
		ID:      id,
		Title:   title,
		Status:  normalizeCriticalCheckStatus(h.Status),
		Details: details,
		Source:  "HBA Scanner",
	}
}

func normalizeCriticalCheckStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pass":
		return "Pass"
	case "fail":
		return "Fail"
	case "manual", "unknown":
		return "Manual"
	default:
		if status == "" {
			return "Manual"
		}
		if strings.EqualFold(status, "Unknown") {
			return "Manual"
		}
		return status
	}
}

func findCISMatch(results []model.Result, parts ...string) *model.Result {
	for i := range results {
		if cisMatchesAll(results[i], parts...) {
			return &results[i]
		}
	}
	return nil
}

func cisMatchesAll(r model.Result, parts ...string) bool {
	blob := cisSearchBlob(r)
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if !strings.Contains(blob, p) {
			return false
		}
	}
	return len(parts) > 0
}

func findHBAMatch(results []model.HBAScannerResult, control int) *model.HBAScannerResult {
	for i := range results {
		if results[i].Control == control {
			return &results[i]
		}
	}
	return nil
}

func evalCriticalPII(ctx criticalEvalContext, id int, title string) CriticalCheckResult {
	rows, _ := decodePIIResults(ctx.pii)
	if len(rows) == 0 {
		return CriticalCheckResult{ID: id, Title: title, Status: "Pass", Details: "No PII findings in latest scan", Source: "PII Scanner"}
	}
	return CriticalCheckResult{
		ID:      id,
		Title:   title,
		Status:  "Fail",
		Details: fmt.Sprintf("%d PII finding(s) in latest scan", len(rows)),
		Source:  "PII Scanner",
	}
}

func evalCriticalPasswordLeak(ctx criticalEvalContext, id int, title string) CriticalCheckResult {
	pm := passwordManagerText(ctx.report)
	if pm != "" {
		return CriticalCheckResult{ID: id, Title: title, Status: "Fail", Details: trimForTable(pm, 120), Source: "Password Audit"}
	}
	if hasLogPasswordLeak(ctx.report) {
		return CriticalCheckResult{ID: id, Title: title, Status: "Fail", Details: "Password leak indicators in pg_log parser", Source: "Log Parser"}
	}
	return CriticalCheckResult{ID: id, Title: title, Status: "Pass", Details: "No leaked password indicators", Source: "Password Audit"}
}

func evalCriticalSSL(ctx criticalEvalContext, id int, title string) CriticalCheckResult {
	for _, r := range ctx.cis {
		blob := strings.ToLower(r.Control + " " + r.Title + " " + r.Description)
		if strings.Contains(blob, "ssl") || strings.Contains(blob, "tls") {
			if strings.EqualFold(r.Status, "Fail") {
				return checkFromCIS(r, id, title)
			}
		}
	}
	if h := findHBAMatch(ctx.hba, 8); h != nil && strings.EqualFold(h.Status, "Fail") {
		return checkFromHBA(*h, id, title)
	}
	for _, r := range ctx.cis {
		blob := strings.ToLower(r.Control + " " + r.Title)
		if strings.Contains(blob, "ssl") || strings.Contains(blob, "tls") {
			return checkFromCIS(r, id, title)
		}
	}
	return CriticalCheckResult{ID: id, Title: title, Status: "Manual", Source: "CIS / HBA"}
}

func evalCriticalSuperuserCount(ctx criticalEvalContext, id int, title string) CriticalCheckResult {
	sec := usersSectionByTitle(ctx.sections, "superuser")
	if sec == nil {
		return CriticalCheckResult{ID: id, Title: title, Status: "Manual", Source: "Users Report"}
	}
	count := 0
	var names []string
	for _, row := range sec.Table.Rows {
		if len(row) == 0 || isSystemRole(row[0]) {
			continue
		}
		count++
		names = append(names, row[0])
	}
	if count <= 3 {
		return CriticalCheckResult{
			ID:      id,
			Title:   title,
			Status:  "Pass",
			Details: fmt.Sprintf("%d superuser role(s) — within limit of 3", count),
			Source:  "Users Report",
		}
	}
	return CriticalCheckResult{
		ID:      id,
		Title:   title,
		Status:  "Fail",
		Details: fmt.Sprintf("%d superuser roles (%s) — limit is 3", count, strings.Join(names, ", ")),
		Source:  "Users Report",
	}
}

func evalCriticalCommonUsers(ctx criticalEvalContext, id int, title string) CriticalCheckResult {
	var found []string
	for _, role := range loginRolesFromUsersList(ctx.sections) {
		if commonDBUsernames[strings.ToLower(role)] {
			found = append(found, role)
		}
	}
	if len(found) == 0 {
		return CriticalCheckResult{ID: id, Title: title, Status: "Pass", Details: "No common usernames among login roles", Source: "Users Report"}
	}
	return CriticalCheckResult{
		ID:      id,
		Title:   title,
		Status:  "Fail",
		Details: "Common usernames: " + strings.Join(found, ", "),
		Source:  "Users Report",
	}
}

func evalCriticalConnLimits(ctx criticalEvalContext, id int, title string) CriticalCheckResult {
	noLimit := humanRoleNamesInSection(ctx.sections, "without connection limits")
	if len(noLimit) == 0 {
		return CriticalCheckResult{ID: id, Title: title, Status: "Pass", Details: "All reviewed roles have connection limits", Source: "Users Report"}
	}
	names := make([]string, 0, len(noLimit))
	for name := range noLimit {
		names = append(names, name)
	}
	return CriticalCheckResult{
		ID:      id,
		Title:   title,
		Status:  "Fail",
		Details: "Roles without connection limits: " + strings.Join(names, ", "),
		Source:  "Users Report",
	}
}

func evalCriticalDefaultPortPostgres(ctx criticalEvalContext, id int, title string) CriticalCheckResult {
	port := "5432"
	if ctx.run != nil && strings.TrimSpace(ctx.run.TargetPort) != "" {
		port = strings.TrimSpace(ctx.run.TargetPort)
	}
	hasPostgres := false
	for _, role := range loginRolesFromUsersList(ctx.sections) {
		if strings.EqualFold(role, "postgres") {
			hasPostgres = true
			break
		}
	}
	if port == "5432" && hasPostgres {
		return CriticalCheckResult{
			ID:      id,
			Title:   title,
			Status:  "Fail",
			Details: "Default port 5432 and login role postgres both in use",
			Source:  "Host Inventory",
		}
	}
	if port == "5432" {
		return CriticalCheckResult{ID: id, Title: title, Status: "Pass", Details: "Non-default admin role naming", Source: "Host Inventory"}
	}
	if hasPostgres {
		return CriticalCheckResult{ID: id, Title: title, Status: "Pass", Details: "Non-default port in use", Source: "Host Inventory"}
	}
	return CriticalCheckResult{ID: id, Title: title, Status: "Pass", Details: "Default port and postgres role not combined", Source: "Host Inventory"}
}

func countCriticalCheckFailures(checks []CriticalCheckResult) int {
	n := 0
	for _, c := range checks {
		if strings.EqualFold(c.Status, "Fail") {
			n++
		}
	}
	return n
}
