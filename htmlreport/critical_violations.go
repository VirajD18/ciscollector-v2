package htmlreport

// CriticalViolationCheck is one of the 25 canonical hardening checks.
type CriticalViolationCheck struct {
	ID      int
	Title   string
	Status  string
	Details string
	Source  string
}

type CriticalViolationsSummary struct {
	Pass    int
	Fail    int
	Manual  int
	Total   int
	PassPct float64
	FailPct float64
}

type CriticalViolationsReport struct {
	Summary CriticalViolationsSummary
	Checks  []CriticalViolationCheck
}

func summarizeCriticalViolations(checks []CriticalViolationCheck) CriticalViolationsSummary {
	summary := CriticalViolationsSummary{Total: len(checks)}
	for _, c := range checks {
		switch c.Status {
		case "Pass":
			summary.Pass++
		case "Fail":
			summary.Fail++
		default:
			summary.Manual++
		}
	}
	if summary.Total > 0 {
		summary.PassPct = float64(summary.Pass) / float64(summary.Total) * 100
		summary.FailPct = float64(summary.Fail) / float64(summary.Total) * 100
	}
	return summary
}

func (h *HtmlReportHelper) RegisterCriticalViolationsReport(checks []CriticalViolationCheck) {
	if h == nil || len(checks) == 0 {
		return
	}
	h.templateData = append(h.templateData, Tab{
		Title: "Critical Violations",
		Body: &CriticalViolationsReport{
			Summary: summarizeCriticalViolations(checks),
			Checks:  checks,
		},
		Priority: 5,
	})
}
