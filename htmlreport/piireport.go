package htmlreport

import "github.com/VirajD18/ciscollector-v2/pkg/piiscanner"

func (h *HtmlReportHelper) RegisterPIIReport(result *piiscanner.DatabasePIIScanOutput) {
	if result == nil {
		return
	}

	if len(result.Data) == 0 {
		return
	}

	h.AddTab("PII Scanner Report", result)
}
