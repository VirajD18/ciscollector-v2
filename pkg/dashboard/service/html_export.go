package service

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/VirajD18/ciscollector-v2/htmlreport"
	"github.com/VirajD18/ciscollector-v2/model"
	"github.com/VirajD18/ciscollector-v2/pkg/reportstore"
	"github.com/VirajD18/ciscollector-v2/pkg/repository"
	"github.com/VirajD18/ciscollector-v2/postgres"
)

// RenderRunHTML builds the full multi-tab KloudDB Shield HTML report from persisted report_json.
func RenderRunHTML(ctx context.Context, repo repository.Repository, run *reportstore.RunRow) string {
	if run == nil {
		return "<!DOCTYPE html><html><body><p>No run data</p></body></html>"
	}
	data, err := renderFullHTMLReport(ctx, repo, run)
	if err != nil || len(data) == 0 {
		return renderSimpleRunHTML(ctx, repo, run)
	}
	return string(data)
}

func renderFullHTMLReport(ctx context.Context, repo repository.Repository, run *reportstore.RunRow) ([]byte, error) {
	helper := htmlreport.NewHtmlReportHelper()
	report := run.Report
	hasTabs := false

	if checks := criticalChecksForRun(run, resolvePIIReport(ctx, repo, run)); len(checks) > 0 {
		helper.RegisterCriticalViolationsReport(criticalChecksForHTML(checks))
		hasTabs = true
	}

	if cis := decodeCISResults(report); len(cis) > 0 {
		ptrs := cisResultPointers(cis)
		helper.RegisterPostgresReportData(ptrs, postgres.CalculateScore(ptrs), decodePostgresVersion(report), report.Host, true)
		hasTabs = true
	}

	if users := decodeUserlistResults(report); len(users) > 0 {
		helper.RegisterUserlistData(users)
		hasTabs = true
	}

	if hba := decodeHBAResults(report); len(hba) > 0 {
		helper.RegisterHBAReportData(hbaResultPointers(hba))
		hasTabs = true
	}

	if ssl := decodeSSLResults(report); ssl != nil {
		helper.RegisterSSLReport(ssl)
		hasTabs = true
	}

	if names := decodePasswordManagerUsernames(report); len(names) > 0 {
		helper.RenderPasswordManagerReport(context.Background(), names)
		hasTabs = true
	}

	if !hasTabs {
		return nil, nil
	}

	helper.CreateAllTab()
	return helper.Render()
}

func cisResultPointers(cis []model.Result) []*model.Result {
	ptrs := make([]*model.Result, len(cis))
	for i := range cis {
		ptrs[i] = &cis[i]
	}
	return ptrs
}

func hbaResultPointers(hba []model.HBAScannerResult) []*model.HBAScannerResult {
	ptrs := make([]*model.HBAScannerResult, len(hba))
	for i := range hba {
		ptrs[i] = &hba[i]
	}
	return ptrs
}

func decodeUserlistResults(report map[string]interface{}) []model.UserlistResult {
	raw, ok := report["Users Report"]
	if !ok {
		return nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var out []model.UserlistResult
	if err := json.Unmarshal(b, &out); err != nil {
		return nil
	}
	return out
}

func decodePasswordManagerUsernames(report map[string]interface{}) []string {
	raw, ok := report["Password Manager Report"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		b, err := json.Marshal(raw)
		if err != nil {
			return nil
		}
		var out []string
		if json.Unmarshal(b, &out) == nil {
			return out
		}
	}
	return nil
}

// renderSimpleRunHTML is a minimal fallback when template rendering is unavailable.
func renderSimpleRunHTML(ctx context.Context, repo repository.Repository, run *reportstore.RunRow) string {
	host := html.EscapeString(hostLabel(run))
	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>KloudDB Shield — ")
	b.WriteString(host)
	b.WriteString("</title><style>body{font-family:system-ui,sans-serif;margin:24px;background:#0f1117;color:#e6e6e6;}table{border-collapse:collapse;width:100%;margin:12px 0;}th,td{border:1px solid #333;padding:8px;text-align:left;font-size:13px;}th{background:#1a1d27;}.fail{color:#f87171;}.pass{color:#4ade80;}h2{margin-top:28px;}</style></head><body>")
	fmt.Fprintf(&b, "<h1>KloudDB Shield — %s</h1><p>Run %s · %s · score %.0f%% · pass %d / fail %d</p>",
		host, html.EscapeString(run.ID), run.StartedAt.Format("2006-01-02 15:04 UTC"),
		run.OverallScore, run.TotalPass, run.TotalFail)

	if checks := criticalChecksForRun(run, resolvePIIReport(ctx, repo, run)); len(checks) > 0 {
		b.WriteString("<h2>Critical Violations</h2><table><thead><tr><th>#</th><th>Violation</th><th>Status</th><th>Source</th><th>Details</th></tr></thead><tbody>")
		for _, c := range checks {
			cls := "pass"
			if strings.EqualFold(c.Status, "Fail") {
				cls = "fail"
			}
			fmt.Fprintf(&b, "<tr><td>%02d</td><td>%s</td><td class=\"%s\">%s</td><td>%s</td><td>%s</td></tr>",
				c.ID, html.EscapeString(c.Title), cls, html.EscapeString(c.Status),
				html.EscapeString(c.Source), html.EscapeString(trimForTable(c.Details, 120)))
		}
		b.WriteString("</tbody></table>")
	}

	cis := decodeCISResults(run.Report)
	if len(cis) > 0 {
		b.WriteString("<h2>CIS / Postgres Report</h2><table><thead><tr><th>Control</th><th>Title</th><th>Status</th><th>Fail reason</th></tr></thead><tbody>")
		for _, r := range cis {
			cls := "pass"
			if strings.EqualFold(r.Status, "Fail") {
				cls = "fail"
			}
			fmt.Fprintf(&b, "<tr><td>%s</td><td>%s</td><td class=\"%s\">%s</td><td>%s</td></tr>",
				html.EscapeString(r.Control), html.EscapeString(trimForTable(r.Title, 80)),
				cls, html.EscapeString(r.Status), html.EscapeString(trimForTable(r.FailReason, 120)))
		}
		b.WriteString("</tbody></table>")
	}

	hba := decodeHBAResults(run.Report)
	if len(hba) > 0 {
		b.WriteString("<h2>HBA Scanner Report</h2><table><thead><tr><th>#</th><th>Check</th><th>Status</th></tr></thead><tbody>")
		for _, h := range hba {
			cls := "pass"
			if strings.EqualFold(h.Status, "Fail") {
				cls = "fail"
			}
			fmt.Fprintf(&b, "<tr><td>%d</td><td>%s</td><td class=\"%s\">%s</td></tr>",
				h.Control, html.EscapeString(h.Title), cls, html.EscapeString(h.Status))
		}
		b.WriteString("</tbody></table>")
	}

	b.WriteString("<p style=\"color:#888;margin-top:32px;\">Generated from SQLite report_json · KloudDB Shield</p></body></html>")
	return b.String()
}

func criticalChecksForHTML(checks []CriticalCheckResult) []htmlreport.CriticalViolationCheck {
	out := make([]htmlreport.CriticalViolationCheck, len(checks))
	for i, c := range checks {
		out[i] = htmlreport.CriticalViolationCheck{
			ID:      c.ID,
			Title:   c.Title,
			Status:  c.Status,
			Details: c.Details,
			Source:  c.Source,
		}
	}
	return out
}
