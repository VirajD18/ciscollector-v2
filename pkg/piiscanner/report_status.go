package piiscanner

import "strings"

const (
	ReportStatusSuccess  = "success"
	ReportStatusNoTables = "no_tables"
	ReportStatusNoData   = "no_data"
	ReportStatusError    = "error"
)

// FinalizeReportJSON adds scan outcome fields to the dashboard payload.
func FinalizeReportJSON(payload map[string]interface{}, status, message, errDetail string) map[string]interface{} {
	if payload == nil {
		payload = map[string]interface{}{}
	}
	if strings.TrimSpace(status) != "" {
		payload["status"] = status
	}
	if strings.TrimSpace(message) != "" {
		payload["message"] = message
	}
	if strings.TrimSpace(errDetail) != "" {
		payload["error"] = errDetail
	}
	return payload
}

// ReportPayload builds the JSON stored in pii_report_json for the dashboard.
func ReportPayload(out *DatabasePIIScanOutput, cnf Config, status, message, errDetail string) map[string]interface{} {
	payload := ToReportJSON(out, cnf)
	switch status {
	case ReportStatusError, ReportStatusNoTables:
		return FinalizeReportJSON(payload, status, message, errDetail)
	}
	if !isEmptyPIIResult(payload) {
		return FinalizeReportJSON(payload, ReportStatusSuccess, message, errDetail)
	}
	if status == "" || status == ReportStatusSuccess {
		status = ReportStatusNoData
		if message == "" {
			message = "No PII data found in database."
		}
	}
	return FinalizeReportJSON(payload, status, message, errDetail)
}

func isEmptyPIIResult(payload map[string]interface{}) bool {
	if len(payload) == 0 {
		return true
	}
	if tableRowCount(payload["high_confidence"]) > 0 {
		return false
	}
	if tableRowCount(payload["meta"]) > 0 {
		return false
	}
	if raw, ok := payload["low_confidence_tables"].([]interface{}); ok && len(raw) > 0 {
		return false
	}
	if raw, ok := payload["low_confidence_tables"].([]string); ok && len(raw) > 0 {
		return false
	}
	return true
}

func tableRowCount(raw interface{}) int {
	t, ok := raw.(map[string]interface{})
	if !ok {
		return 0
	}
	rows, ok := t["rows"].([]interface{})
	if !ok {
		return 0
	}
	return len(rows)
}
