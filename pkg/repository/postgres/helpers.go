package postgres

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/klouddb/klouddbshield/pkg/postgresdb"
	"github.com/klouddb/klouddbshield/pkg/reportstore"
)

func rebindPostgres(query string) string {
	var b strings.Builder
	n := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			n++
		} else {
			b.WriteByte(query[i])
		}
	}
	return b.String()
}

const runSelectColumns = `id, started_at, finished_at, trigger, runner_name,
		target_type, target_id, target_host, target_port, target_db,
		run_status, features_run, overall_score, total_pass, total_fail, report_json,
		pii_report_json, pii_scanned_at`

func encodeReport(fileData map[string]interface{}) ([]byte, error) {
	raw, err := json.Marshal(fileData)
	if err != nil {
		return nil, fmt.Errorf("marshal report: %w", err)
	}
	return raw, nil
}

func decodeReport(blob []byte) (map[string]interface{}, error) {
	if len(blob) == 0 {
		return map[string]interface{}{}, nil
	}
	if blob[0] == '{' || blob[0] == '[' {
		var out map[string]interface{}
		if err := json.Unmarshal(blob, &out); err != nil {
			return nil, fmt.Errorf("unmarshal report: %w", err)
		}
		return out, nil
	}
	zr, err := gzip.NewReader(bytes.NewReader(blob))
	if err != nil {
		return nil, fmt.Errorf("gzip open: %w", err)
	}
	defer zr.Close()
	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("gzip read: %w", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal report: %w", err)
	}
	return out, nil
}

func targetFields(pg *postgresdb.Postgres) (host, port, db string) {
	if pg == nil {
		return "unknown", "5432", "postgres"
	}
	host = reportstore.NormalizeHost(pg.Host)
	port = pg.Port
	if port == "" {
		port = "5432"
	}
	db = pg.DBName
	if db == "" {
		db = "postgres"
	}
	return host, port, db
}

func mergeFeaturesRun(existingJSON, feature string) string {
	var features []string
	if existingJSON != "" {
		_ = json.Unmarshal([]byte(existingJSON), &features)
	}
	for _, f := range features {
		if f == feature {
			return existingJSON
		}
	}
	features = append(features, feature)
	out, _ := json.Marshal(features)
	return string(out)
}

func summarizeFromReport(fileData map[string]interface{}) (pass, fail int, score float64) {
	pr, ok := fileData["Postgres Report"].(map[string]interface{})
	if !ok {
		return 0, 0, 0
	}
	if sc, ok := pr["score"].(map[string]interface{}); ok {
		if v, ok := sc["Pass"].(float64); ok {
			pass = int(v)
		} else if v, ok := sc["Pass"].(int); ok {
			pass = v
		}
		if v, ok := sc["Fail"].(float64); ok {
			fail = int(v)
		} else if v, ok := sc["Fail"].(int); ok {
			fail = v
		}
	}
	if pass+fail == 0 {
		pass, fail = countPassFailFromResult(pr["result"])
	}
	total := pass + fail
	if total > 0 {
		score = float64(pass) / float64(total) * 100
	}
	return pass, fail, score
}

func countPassFailFromResult(raw interface{}) (pass, fail int) {
	if raw == nil {
		return 0, 0
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return 0, 0
	}
	var rows []struct {
		Status string `json:"Status"`
	}
	if err := json.Unmarshal(b, &rows); err != nil {
		return 0, 0
	}
	for _, r := range rows {
		if strings.EqualFold(r.Status, "Pass") {
			pass++
		} else if strings.EqualFold(r.Status, "Fail") {
			fail++
		}
	}
	return pass, fail
}

func scanRunRow(row *sql.Row) (*reportstore.RunRow, error) {
	var (
		id, started, finished, trigger, runner, ttype, tid, host, port, dbname, status string
		featuresJSON                                                                   sql.NullString
		score                                                                          sql.NullFloat64
		pass, fail                                                                     sql.NullInt64
		blob                                                                           []byte
		piiBlob                                                                        []byte
		piiScanned                                                                     sql.NullString
	)
	if err := row.Scan(&id, &started, &finished, &trigger, &runner, &ttype, &tid, &host, &port, &dbname,
		&status, &featuresJSON, &score, &pass, &fail, &blob, &piiBlob, &piiScanned); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return buildRunRow(id, started, finished, trigger, runner, ttype, tid, host, port, dbname, status,
		featuresJSON, score, pass, fail, blob, piiBlob, piiScanned)
}

func scanRunsFromRows(rows *sql.Rows) ([]reportstore.RunRow, error) {
	var out []reportstore.RunRow
	for rows.Next() {
		var (
			id, started, finished, trigger, runner, ttype, tid, host, port, dbname, status string
			featuresJSON                                                                   sql.NullString
			score                                                                          sql.NullFloat64
			pass, fail                                                                     sql.NullInt64
			blob                                                                           []byte
			piiBlob                                                                        []byte
			piiScanned                                                                     sql.NullString
		)
		if err := rows.Scan(&id, &started, &finished, &trigger, &runner, &ttype, &tid, &host, &port, &dbname,
			&status, &featuresJSON, &score, &pass, &fail, &blob, &piiBlob, &piiScanned); err != nil {
			return nil, err
		}
		r, err := buildRunRow(id, started, finished, trigger, runner, ttype, tid, host, port, dbname, status,
			featuresJSON, score, pass, fail, blob, piiBlob, piiScanned)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func buildRunRow(id, started, finished, trigger, runner, ttype, tid, host, port, dbname, status string,
	featuresJSON sql.NullString, score sql.NullFloat64, pass, fail sql.NullInt64,
	blob, piiBlob []byte, piiScanned sql.NullString) (*reportstore.RunRow, error) {
	st, _ := time.Parse(time.RFC3339, started)
	ft, _ := time.Parse(time.RFC3339, finished)
	report, err := decodeReport(blob)
	if err != nil {
		return nil, err
	}
	piiReport, err := decodeReport(piiBlob)
	if err != nil {
		return nil, err
	}
	var features []string
	if featuresJSON.Valid && featuresJSON.String != "" {
		_ = json.Unmarshal([]byte(featuresJSON.String), &features)
	}
	r := &reportstore.RunRow{
		ID: id, StartedAt: st, FinishedAt: ft, Trigger: trigger, RunnerName: runner,
		TargetType: ttype, TargetID: tid, TargetHost: host, TargetPort: port, TargetDB: dbname,
		RunStatus: status, FeaturesRun: features, Report: report, PiiReport: piiReport,
	}
	if piiScanned.Valid && piiScanned.String != "" {
		r.PiiScannedAt, _ = time.Parse(time.RFC3339, piiScanned.String)
	}
	if score.Valid {
		r.OverallScore = score.Float64
	}
	if pass.Valid {
		r.TotalPass = int(pass.Int64)
	}
	if fail.Valid {
		r.TotalFail = int(fail.Int64)
	}
	return r, nil
}
