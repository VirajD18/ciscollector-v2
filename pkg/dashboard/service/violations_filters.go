package service

import (
	"context"
	"sort"
)

// fleetCategoryViolationTypes maps fleet tile IDs to drill-down violation type labels.
// Kept in sync with fleet category IDs in fleet.go (scanner catalog, not demo rows).
var fleetCategoryViolationTypes = map[string]string{
	"pii-violations":   "PII Exposure",
	"ssl-violations":   "SSL Violation",
	"password-leakage": "Password Leak",
	"config-audit":     "Critical Config",
	"elevated-privs":   "Unauthorized Superuser",
	"hba-policy":       "HBA Violation",
}

func (s *Service) buildViolationFilterOptions(ctx context.Context, rows []CriticalViolationRow) (typeOptions, severityOptions []string) {
	seenTypes := map[string]bool{}
	addType := func(t string) {
		if t != "" {
			seenTypes[t] = true
		}
	}

	if cats, err := s.FleetCategories(ctx); err == nil && cats != nil {
		for _, c := range cats.Categories {
			if label, ok := fleetCategoryViolationTypes[c.ID]; ok {
				addType(label)
			}
		}
	}
	for _, r := range rows {
		addType(r.Type)
	}

	typeOptions = make([]string, 0, len(seenTypes))
	for t := range seenTypes {
		typeOptions = append(typeOptions, t)
	}
	sort.Strings(typeOptions)

	seenSev := map[string]bool{"CRITICAL": true, "HIGH": true}
	for _, r := range rows {
		if r.Severity != "" {
			seenSev[r.Severity] = true
		}
	}
	severityOptions = make([]string, 0, 2)
	for _, sev := range []string{"CRITICAL", "HIGH"} {
		if seenSev[sev] {
			severityOptions = append(severityOptions, sev)
		}
	}
	return typeOptions, severityOptions
}
