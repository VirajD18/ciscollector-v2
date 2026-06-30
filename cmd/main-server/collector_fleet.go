package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/VirajD18/ciscollector-v2/pkg/config"
)

type collectorRegisterRequest struct {
	SchemaVersion string    `json:"schema_version"`
	NodeID        string    `json:"node_id"`
	Hostname      string    `json:"hostname"`
	IP            string    `json:"ip,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
	Schedule      string    `json:"schedule,omitempty"`
	ScanCommands  string    `json:"scan_commands,omitempty"`
	ScheduledJobs int       `json:"scheduled_jobs"`
}

func (a *App) resolveKshieldConfigPath() string {
	if p := strings.TrimSpace(a.KshieldConfigPath); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	const systemPath = "/etc/klouddbshield/kshieldconfig.toml"
	if _, err := os.Stat(systemPath); err == nil {
		return systemPath
	}
	if wd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(wd, "kshieldconfig.toml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func (a *App) collectorOfflineThreshold() time.Duration {
	sec := config.OfflineThresholdFromPath(a.resolveKshieldConfigPath())
	return time.Duration(sec) * time.Second
}

func (a *App) collectorPushIntervalSec() int {
	path := a.resolveKshieldConfigPath()
	c, err := config.LoadFromPath(path)
	if err != nil || c == nil {
		return 30
	}
	d := c.MainServer.EffectivePushInterval()
	if d <= 0 {
		return 30
	}
	return int(d.Seconds())
}

func (a *App) broadcastCollectorFleetEvent(eventType string, view CollectorNodeView) {
	if a.WSHub == nil {
		return
	}
	a.WSHub.BroadcastMessage(WSMessage{
		Type: eventType,
		Payload: map[string]any{
			"node_id":        view.NodeID,
			"hostname":       view.Hostname,
			"status":         view.Status,
			"scheduled_jobs": view.ScheduledJobs,
			"last_seen_at":   view.LastSeenAt,
		},
	})
}

func (a *App) collectorRegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req collectorRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.SchemaVersion != collectorSchemaVersion {
		http.Error(w, "unsupported schema", http.StatusBadRequest)
		return
	}
	nodeID := strings.TrimSpace(req.NodeID)
	hostname := strings.TrimSpace(req.Hostname)
	if nodeID == "" || hostname == "" {
		http.Error(w, "node_id and hostname required", http.StatusBadRequest)
		return
	}
	ts := req.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	ctx := r.Context()
	if err := requireService(a); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	msg := "collector registered"
	if sched := strings.TrimSpace(req.Schedule); sched != "" {
		msg += " (schedule=" + sched + ")"
	}
	if cmds := strings.TrimSpace(req.ScanCommands); cmds != "" {
		msg += " commands=" + cmds
	}
	if err := a.Svc.RegisterCollector(ctx, nodeID, hostname, req.IP, ts, req.ScheduledJobs, msg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	view := CollectorNodeView{
		NodeID:        nodeID,
		Hostname:      hostname,
		IP:            strings.TrimSpace(req.IP),
		Status:        "online",
		ScheduledJobs: req.ScheduledJobs,
		LastSeenAt:    ts,
	}
	a.broadcastCollectorFleetEvent("collector_registered", view)

	writeJSON(w, http.StatusOK, map[string]any{
		"registered":    true,
		"node_id":       nodeID,
		"hostname":      hostname,
		"registered_at": ts,
	})
}
