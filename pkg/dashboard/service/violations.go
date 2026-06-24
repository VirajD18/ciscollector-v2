package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Violations returns critical/high findings from latest SQLite scan runs.
func (s *Service) Violations(ctx context.Context) (*ViolationsResponse, error) {
	runs, err := s.latestRunsByTarget(ctx)
	if err != nil {
		return nil, err
	}
	resp := &ViolationsResponse{
		Critical: []ViolationEntry{},
		High:     []ViolationEntry{},
		Medium:   []ViolationEntry{},
	}
	for _, run := range runs {
		host := hostLabel(run)
		detected := formatDetectedAt(run.StartedAt)

		for _, r := range decodeCISResults(run.Report) {
			if !strings.EqualFold(r.Status, "Fail") {
				continue
			}
			vtype := violationTypeFromCIS(r)
			sev := "high"
			if r.Critical {
				sev = "critical"
			}
			resp.Critical = append(resp.Critical, ViolationEntry{
				Host:          host,
				Check:         violationDetailsFromCIS(r),
				Severity:      sev,
				ViolationType: vtype,
				DetectedAt:    detected,
			})
		}
		for _, h := range decodeHBAResults(run.Report) {
			if !strings.EqualFold(h.Status, "Fail") {
				continue
			}
			check := h.Title
			if check == "" {
				check = fmt.Sprintf("HBA check %d", h.Control)
			}
			resp.High = append(resp.High, ViolationEntry{
				Host:          host,
				Check:         check,
				Severity:      "high",
				ViolationType: "HBA Violation",
				DetectedAt:    detected,
			})
		}
		appendSuperuserViolations(run.Report, host, detected, resp)
	}
	resp.Rows = buildCriticalRowsFromResponse(resp)
	resp.TypeOptions, resp.SeverityOptions = s.buildViolationFilterOptions(ctx, resp.Rows)
	return resp, nil
}

func formatDetectedAt(t time.Time) string {
	if t.IsZero() {
		return time.Now().UTC().Format("2006-01-02 15:04")
	}
	return t.UTC().Format("2006-01-02 15:04")
}

func appendSuperuserViolations(report map[string]interface{}, host, detected string, resp *ViolationsResponse) {
	sections := decodeUsersReportSections(report)
	sec := usersSectionByTitle(sections, "superuser")
	if sec == nil {
		return
	}
	for _, row := range sec.Table.Rows {
		if len(row) == 0 || isSystemRole(row[0]) {
			continue
		}
		role := row[0]
		resp.Critical = append(resp.Critical, ViolationEntry{
			Host:          host,
			Check:         role + " has SUPERUSER (Users Report)",
			Severity:      "high",
			ViolationType: "Unauthorized Superuser",
			DetectedAt:    detected,
		})
	}
}

func buildCriticalRowsFromResponse(v *ViolationsResponse) []CriticalViolationRow {
	if v == nil {
		return nil
	}
	var rows []CriticalViolationRow
	id := 1
	appendRow := func(e ViolationEntry, defaultType string) {
		sev := "HIGH"
		if e.Severity == "critical" {
			sev = "CRITICAL"
		}
		vtype := e.ViolationType
		if vtype == "" {
			vtype = defaultType
		}
		rows = append(rows, CriticalViolationRow{
			ID:            fmt.Sprintf("V-%03d", id),
			Server:        e.Host,
			Type:          vtype,
			Details:       e.Check,
			Severity:      sev,
			Detected:      e.DetectedAt,
			Status:        "Open",
			ConfigSection: configSectionForType(vtype),
		})
		id++
	}
	for _, c := range v.Critical {
		appendRow(c, "Critical Config")
	}
	for _, h := range v.High {
		appendRow(h, "HBA Violation")
	}
	return rows
}

// CriticalViolationRows expands violations for the critical-violations table API.
func (s *Service) CriticalViolationRows(ctx context.Context) ([]CriticalViolationRow, error) {
	v, err := s.Violations(ctx)
	if err != nil {
		return nil, err
	}
	return v.Rows, nil
}
