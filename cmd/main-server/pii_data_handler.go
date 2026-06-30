package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
)

// PiiReportRequest is the PII payload from ciscollector.
type PiiReportRequest struct {
	SchemaVersion string                 `json:"schema_version"`
	Node          NodeInfo               `json:"node"`
	Timestamp     time.Time              `json:"timestamp"`
	TargetHost    string                 `json:"target_host"`
	TargetPort    string                 `json:"target_port"`
	TargetDB      string                 `json:"target_db"`
	PiiReport     map[string]interface{} `json:"pii_report"`
	ScannedAt     time.Time              `json:"scanned_at"`
}

func (a *App) piiDataPostHandler(w http.ResponseWriter, r *http.Request) {
	body, err := readRequestBody(r)
	if err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	var req PiiReportRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.SchemaVersion != "v1" {
		http.Error(w, "unsupported schema", http.StatusBadRequest)
		return
	}
	if !collectorBodyTokenAllowed(req.Node.AgentConfig.Server.Token, a.ServerConfig.Token) {
		http.Error(w, "Token not matched", http.StatusBadRequest)
		return
	}
	if len(req.PiiReport) == 0 {
		http.Error(w, "pii_report is required", http.StatusBadRequest)
		return
	}

	if a.Svc == nil {
		http.Error(w, "main database not initialized", http.StatusInternalServerError)
		return
	}

	port := req.TargetPort
	if port == "" {
		port = "5432"
	}
	dbName := req.TargetDB
	if dbName == "" {
		dbName = "postgres"
	}
	host := req.TargetHost
	if host == "" {
		host = req.Node.IP
	}

	pg := &postgresdb.Postgres{
		Host:   host,
		Port:   port,
		DBName: dbName,
	}
	if err := a.Svc.PersistPIIReport(r.Context(), pg, req.PiiReport); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
