package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/klouddb/klouddbshield/model"
	"github.com/klouddb/klouddbshield/pkg/logparser"
)

// LogReadinessHostRow is one host in the GUC readiness fleet table.
type LogReadinessHostRow struct {
	Host               string `json:"host"`
	LogConnections     string `json:"logConnections"`
	LogConnectionsOK   bool   `json:"logConnectionsOk"`
	LogLinePrefix      string `json:"logLinePrefix"`
	LogLinePrefixCIS   string `json:"logLinePrefixCis"`
	LogParserReadiness string `json:"logparserReadiness"`
}

// LogReadinessFleetResponse is fleet GUC readiness from latest scans.
type LogReadinessFleetResponse struct {
	Rows    []LogReadinessHostRow `json:"rows"`
	Message string                `json:"message,omitempty"`
}

// LogReadinessFleet returns per-host GUC readiness from report_json.
func (s *Service) LogReadinessFleet(ctx context.Context) (*LogReadinessFleetResponse, error) {
	runs, err := s.latestRunsByTarget(ctx)
	if err != nil {
		return nil, err
	}
	resp := &LogReadinessFleetResponse{Rows: []LogReadinessHostRow{}}
	for _, run := range runs {
		if run == nil || run.Report == nil {
			continue
		}
		row, ok := logReadinessRowFromReport(run.Report)
		if !ok {
			row = logReadinessRowFallback(run.Report)
		}
		row.Host = hostLabel(run)
		resp.Rows = append(resp.Rows, row)
	}
	if len(resp.Rows) == 0 {
		resp.Message = "No scan data yet. Run collector with log parser or postgres_cis to populate GUC readiness."
	}
	return resp, nil
}

func logReadinessRowFromReport(report map[string]interface{}) (LogReadinessHostRow, bool) {
	raw, ok := report[logparser.LogReadinessReportKey]
	if !ok {
		return LogReadinessHostRow{}, false
	}
	m, ok := raw.(map[string]interface{})
	if !ok {
		m = decodeJSONMap(raw)
	}
	if m == nil {
		return LogReadinessHostRow{}, false
	}
	return logReadinessRowFromMap(m), true
}

func logReadinessRowFallback(report map[string]interface{}) LogReadinessHostRow {
	connOn, prefix := gucLogSettingsFromReport(report)
	if prefix == "" && !connOn {
		if cis := decodeCISResults(report); len(cis) > 0 {
			connOn, prefix = cisLogSettings(cis)
		}
	}
	in := logparser.ReadinessInput{
		LogConnectionsOn: connOn,
		LogLinePrefix:    prefix,
	}
	return logReadinessRowFromMap(logparser.BuildReadinessReport(in))
}

func logReadinessRowFromMap(m map[string]interface{}) LogReadinessHostRow {
	if m == nil {
		return LogReadinessHostRow{}
	}
	conn := strings.TrimSpace(stringField(m, "log_connections"))
	connOn := boolField(m, "log_connections_on")
	if !connOn && strings.EqualFold(conn, "on") {
		connOn = true
	}
	prefix := stringField(m, "log_line_prefix")
	prefixCIS := stringField(m, "log_line_prefix_cis", "logLinePrefixCis")
	readiness := stringField(m, "logparser_readiness", "logparserReadiness")
	if readiness == "" {
		readiness = logParserReadinessFromRunnable(m)
	}
	if prefix == "" {
		prefix = "—"
	}
	return LogReadinessHostRow{
		LogConnections:     conn,
		LogConnectionsOK:   connOn,
		LogLinePrefix:      prefix,
		LogLinePrefixCIS:   prefixCIS,
		LogParserReadiness: readiness,
	}
}

func logParserReadinessFromRunnable(m map[string]interface{}) string {
	raw, ok := m["runnable_commands"]
	if !ok {
		return "FAIL"
	}
	switch t := raw.(type) {
	case []string:
		if len(t) > 0 {
			return "PASS"
		}
	case []interface{}:
		if len(t) > 0 {
			return "PASS"
		}
	}
	return "FAIL"
}

func gucLogSettingsFromReport(report map[string]interface{}) (connOn bool, prefix string) {
	raw, ok := report["GUC Settings"]
	if !ok {
		return false, ""
	}
	block, ok := raw.(map[string]interface{})
	if !ok {
		block = decodeJSONMap(raw)
	}
	settingsRaw, ok := block["settings"]
	if !ok {
		return false, ""
	}
	settings := map[string]string{}
	b, err := json.Marshal(settingsRaw)
	if err != nil {
		return false, ""
	}
	if err := json.Unmarshal(b, &settings); err != nil {
		return false, ""
	}
	conn := strings.TrimSpace(settings["log_connections"])
	connOn = strings.EqualFold(conn, "on") || strings.EqualFold(conn, "yes")
	return connOn, settings["log_line_prefix"]
}

func cisLogSettings(cis []model.Result) (connOn bool, prefix string) {
	for _, r := range cis {
		title := strings.ToLower(r.Title)
		if strings.Contains(title, "log_connections") {
			connOn = strings.EqualFold(r.Status, "Pass")
		}
		if strings.Contains(title, "log_line_prefix") {
			vals := gucValuesFromManualCheck(r)
			if v, ok := vals["log_line_prefix"]; ok {
				prefix = v
			}
		}
	}
	return connOn, prefix
}

func decodeJSONMap(raw interface{}) map[string]interface{} {
	if raw == nil {
		return nil
	}
	if m, ok := raw.(map[string]interface{}); ok {
		return m
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	out := map[string]interface{}{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil
	}
	return out
}

func boolField(m map[string]interface{}, keys ...string) bool {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case bool:
				return t
			case string:
				return strings.EqualFold(t, "true") || strings.EqualFold(t, "on")
			}
		}
	}
	return false
}
