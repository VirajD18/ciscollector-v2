package htmlreport

import "github.com/VirajD18/ciscollector-v2/postgres/calctransactions"

func (h *HtmlReportHelper) RegisterCalcTranx(data calctransactions.ReportData) {
	h.AddTab("Wraparound Report", data)
}
