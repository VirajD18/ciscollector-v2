package postgres

import (
	"database/sql"
	"encoding/json"

	reprows "github.com/VirajD18/ciscollector-v2/pkg/repository/rows"
)

func scanCollectorNodes(rows *sql.Rows) ([]reprows.CollectorNodeRow, error) {
	var out []reprows.CollectorNodeRow
	for rows.Next() {
		n, err := scanCollectorFromRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *n)
	}
	return out, rows.Err()
}

func scanCollectorNodeRow(row *sql.Row) (*reprows.CollectorNodeRow, error) {
	var v reprows.CollectorNodeRow
	var cronRunning int
	var lastRun sql.NullTime
	var lastErr sql.NullString
	var ip sql.NullString
	if err := row.Scan(&v.NodeID, &v.Hostname, &ip, &v.LastSeenAt, &cronRunning, &v.ScheduledJobs,
		&lastRun, &lastErr); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if ip.Valid {
		v.IP = ip.String
	}
	v.CronRunning = cronRunning == 1
	if lastRun.Valid {
		t := lastRun.Time
		v.LastRunAt = &t
	}
	if lastErr.Valid {
		v.LastError = lastErr.String
	}
	return &v, nil
}

func scanCollectorFromRows(rows *sql.Rows) (*reprows.CollectorNodeRow, error) {
	var v reprows.CollectorNodeRow
	var cronRunning int
	var lastRun sql.NullTime
	var lastErr sql.NullString
	var ip sql.NullString
	if err := rows.Scan(&v.NodeID, &v.Hostname, &ip, &v.LastSeenAt, &cronRunning, &v.ScheduledJobs,
		&lastRun, &lastErr); err != nil {
		return nil, err
	}
	if ip.Valid {
		v.IP = ip.String
	}
	v.CronRunning = cronRunning == 1
	if lastRun.Valid {
		t := lastRun.Time
		v.LastRunAt = &t
	}
	if lastErr.Valid {
		v.LastError = lastErr.String
	}
	return &v, nil
}

func scanCollectorRuns(rows *sql.Rows) ([]reprows.CollectorRunRow, error) {
	var out []reprows.CollectorRunRow
	for rows.Next() {
		var row reprows.CollectorRunRow
		var finished sql.NullTime
		var features sql.NullString
		var success int
		var errMsg sql.NullString
		if err := rows.Scan(&row.ID, &row.Trigger, &row.StartedAt, &finished, &features, &success, &errMsg); err != nil {
			return nil, err
		}
		if finished.Valid {
			t := finished.Time
			row.FinishedAt = &t
		}
		if features.Valid && features.String != "" {
			_ = json.Unmarshal([]byte(features.String), &row.Features)
		}
		row.Success = success == 1
		if errMsg.Valid {
			row.Error = errMsg.String
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
