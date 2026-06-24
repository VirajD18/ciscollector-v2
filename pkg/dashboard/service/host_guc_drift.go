package service

import (
	"context"
	"strings"

	"github.com/klouddb/klouddbshield/postgresconfig"
)

func (s *Service) resolveGucSnapshotTargetID(ctx context.Context, serverID, runTargetID string) string {
	if s == nil || s.Repo == nil {
		return runTargetID
	}
	serverID = strings.TrimSpace(serverID)
	snapshots, err := s.Repo.ListServerGucSnapshots(ctx)
	if err != nil {
		return runTargetID
	}
	for _, snap := range snapshots {
		if serverID != "" && strings.EqualFold(gucSnapshotHostLabel(snap), serverID) {
			return snap.TargetID
		}
	}
	if runTargetID != "" {
		for _, snap := range snapshots {
			if snap.TargetID == runTargetID {
				return runTargetID
			}
		}
	}
	return runTargetID
}

func (s *Service) buildHostGucDriftView(ctx context.Context, serverID, targetID string) HostGucDriftView {
	out := HostGucDriftView{Available: false, Status: "no_baseline"}
	if s == nil || s.Repo == nil {
		out.EmptyReason = "Database not configured."
		return out
	}
	label, baseline, _, err := s.Repo.GetGucBaseline(ctx)
	if err != nil {
		out.EmptyReason = "Failed to load baseline."
		return out
	}
	if len(baseline) == 0 {
		out.EmptyReason = "Upload a global baseline on the GUC drift page."
		return out
	}
	out.BaselineLabel = label
	out.Available = true

	targetID = strings.TrimSpace(targetID)
	if targetID == "" {
		out.Status = "no_snapshot"
		out.EmptyReason = "No SHOW ALL snapshot for this host yet."
		return out
	}

	live, host, _, err := s.Repo.GetServerGucSnapshot(ctx, targetID)
	if err != nil {
		out.Status = "no_snapshot"
		out.EmptyReason = "Failed to load GUC snapshot."
		return out
	}
	if len(live) == 0 {
		out.Status = "no_snapshot"
		out.EmptyReason = "No SHOW ALL snapshot for this host yet."
		return out
	}
	if host == "" {
		host = strings.TrimSpace(serverID)
	}

	rows, driftCount, missingCount := gucDriftRowsForHost(host, targetID, baseline, live)
	out.DriftCount = driftCount
	out.MissingCount = missingCount
	out.Rows = rows
	if driftCount == 0 && missingCount == 0 {
		out.Status = "matched"
		out.EmptyReason = "All baseline keys match live SHOW ALL."
		return out
	}
	if driftCount > 0 {
		out.Status = "drifted"
	} else {
		out.Status = "missing"
	}
	return out
}

func gucDriftRowsForHost(host, targetID string, baseline, live map[string]string) ([]GucDriftRow, int, int) {
	var rows []GucDriftRow
	driftCount, missingCount := 0, 0
	for _, row := range postgresconfig.CompareAgainstBaseline(baseline, live) {
		switch row.Status {
		case postgresconfig.DriftDiff:
			driftCount++
			rows = append(rows, GucDriftRow{
				Host: host, TargetID: targetID, Guc: row.GUC,
				Live: row.Live, Baseline: row.Baseline, Status: string(row.Status),
			})
		case postgresconfig.DriftMissing:
			missingCount++
			rows = append(rows, GucDriftRow{
				Host: host, TargetID: targetID, Guc: row.GUC,
				Live: row.Live, Baseline: row.Baseline, Status: string(row.Status),
			})
		}
	}
	return rows, driftCount, missingCount
}
