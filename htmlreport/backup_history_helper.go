package htmlreport

import "github.com/VirajD18/ciscollector-v2/pkg/backuphistory"

func (h *HtmlReportHelper) RegisterBackupHistory(output backuphistory.BackupHistoryOutput) {
	h.AddTab("Backup Audit Tool", output)
}
