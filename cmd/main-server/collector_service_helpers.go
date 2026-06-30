package main

import (
	"database/sql"
	"time"

	"github.com/VirajD18/ciscollector-v2/pkg/config"
	"github.com/VirajD18/ciscollector-v2/pkg/repository"
)

func connectionStatus(lastSeen time.Time, threshold time.Duration) string {
	if threshold <= 0 {
		threshold = time.Duration(config.DefaultOfflineThresholdSec) * time.Second
	}
	if time.Since(lastSeen) <= threshold {
		return "online"
	}
	return "offline"
}

func collectorNodeViewFromRow(row repository.CollectorNodeRow, threshold time.Duration) CollectorNodeView {
	return CollectorNodeView{
		NodeID:        row.NodeID,
		Hostname:      row.Hostname,
		IP:            row.IP,
		Status:        connectionStatus(row.LastSeenAt, threshold),
		CronRunning:   row.CronRunning,
		ScheduledJobs: row.ScheduledJobs,
		LastSeenAt:    row.LastSeenAt,
		LastRunAt:     row.LastRunAt,
		LastError:     row.LastError,
	}
}

func requireService(a *App) error {
	if a == nil || a.Svc == nil {
		return sql.ErrConnDone
	}
	return nil
}
