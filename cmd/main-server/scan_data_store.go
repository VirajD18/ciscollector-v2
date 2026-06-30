package main

import (
	"context"
	"fmt"
	"time"

	mainserversvc "github.com/VirajD18/ciscollector-v2/pkg/mainserver/service"
)

// ScanDataRequest is the v2 collector scan payload (mirrors mainserverclient.ScanDataRequest).
type ScanDataRequest struct {
	SchemaVersion string                 `json:"schema_version"`
	Node          NodeInfo               `json:"node"`
	Timestamp     time.Time              `json:"timestamp"`
	Data          NodeData               `json:"data"`
	ScanRun       ScanRunMeta            `json:"scan_run"`
	Report        map[string]interface{} `json:"report"`
}

type ScanRunMeta struct {
	Trigger      string    `json:"trigger"`
	Features     []string  `json:"features,omitempty"`
	TargetID     string    `json:"target_id"`
	TargetHost   string    `json:"target_host"`
	TargetPort   string    `json:"target_port"`
	TargetDB     string    `json:"target_db"`
	RunStatus    string    `json:"run_status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
}

func (a *App) storeScanResult(ctx context.Context, req *ScanDataRequest) error {
	if a.Svc == nil {
		return fmt.Errorf("main database not initialized")
	}
	return a.Svc.StoreScanResult(ctx, mainserversvc.ScanDataRequest{
		SchemaVersion: req.SchemaVersion,
		Node: mainserversvc.ScanNodeInfo{
			ID:   req.Node.ID,
			Name: req.Node.Name,
		},
		Timestamp: req.Timestamp,
		Data: mainserversvc.ScanNodeData{
			GucSettings: gucPayloadFromNode(req.Data.GucSettings),
		},
		ScanRun: mainserversvc.ScanRunMeta{
			Trigger:      req.ScanRun.Trigger,
			Features:     req.ScanRun.Features,
			TargetHost:   req.ScanRun.TargetHost,
			TargetPort:   req.ScanRun.TargetPort,
			TargetDB:     req.ScanRun.TargetDB,
			RunStatus:    req.ScanRun.RunStatus,
			ErrorMessage: req.ScanRun.ErrorMessage,
			StartedAt:    req.ScanRun.StartedAt,
			FinishedAt:   req.ScanRun.FinishedAt,
		},
		Report: req.Report,
	})
}

func gucPayloadFromNode(in *GucSettingsPayload) *mainserversvc.GucSettingsPayload {
	if in == nil {
		return nil
	}
	return &mainserversvc.GucSettingsPayload{Settings: in.Settings}
}
