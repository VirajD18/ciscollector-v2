package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/klouddb/klouddbshield/pkg/reportstore"
	"github.com/klouddb/klouddbshield/postgresconfig"
)

func (s *Service) gucDriftFromSnapshots(ctx context.Context) (*GucDriftResponse, error) {
	if s.Repo == nil {
		return &GucDriftResponse{}, nil
	}
	label, baseline, _, err := s.Repo.GetGucBaseline(ctx)
	if err != nil {
		return nil, err
	}
	snapshots, err := s.Repo.ListServerGucSnapshots(ctx)
	if err != nil {
		return nil, err
	}

	resp := &GucDriftResponse{
		Stats: GucDriftStats{
			BaselineLabel: label,
			BaselineKeys:  len(baseline),
		},
		HostSummaries: []GucDriftHostSummary{},
		Rows:          []GucDriftRow{},
	}

	if len(baseline) == 0 {
		return resp, nil
	}

	for _, snap := range snapshots {
		resp.Stats.HostsCompared++
		live, _, _, err := s.Repo.GetServerGucSnapshot(ctx, snap.TargetID)
		if err != nil {
			return nil, err
		}
		host := gucSnapshotHostLabel(snap)
		if len(live) == 0 {
			resp.HostSummaries = append(resp.HostSummaries, GucDriftHostSummary{
				Host:     host,
				TargetID: snap.TargetID,
				Status:   "no_snapshot",
			})
			continue
		}

		rows := postgresconfig.CompareAgainstBaseline(baseline, live)
		summary := GucDriftHostSummary{
			Host:     host,
			TargetID: snap.TargetID,
			Status:   "matched",
		}
		for _, row := range rows {
			switch row.Status {
			case postgresconfig.DriftDiff:
				summary.DriftCount++
				resp.Stats.TotalDrifted++
				resp.Rows = append(resp.Rows, GucDriftRow{
					Host:     host,
					TargetID: snap.TargetID,
					Guc:      row.GUC,
					Live:     row.Live,
					Baseline: row.Baseline,
					Status:   string(row.Status),
				})
			case postgresconfig.DriftMissing:
				summary.MissingCount++
				resp.Stats.TotalMissing++
				resp.Rows = append(resp.Rows, GucDriftRow{
					Host:     host,
					TargetID: snap.TargetID,
					Guc:      row.GUC,
					Live:     row.Live,
					Baseline: row.Baseline,
					Status:   string(row.Status),
				})
			}
		}
		if summary.DriftCount == 0 && summary.MissingCount == 0 {
			summary.Status = "matched"
			resp.Stats.MatchedServers++
		} else {
			if summary.DriftCount > 0 {
				summary.Status = "drifted"
				resp.Stats.DriftingServers++
			} else {
				summary.Status = "missing"
			}
			if summary.MissingCount > 0 {
				resp.Stats.MissingServers++
			}
		}
		resp.HostSummaries = append(resp.HostSummaries, summary)
	}
	return resp, nil
}

func hostGucDriftCount(ctx context.Context, s *Service, targetID string) string {
	if s == nil || s.Repo == nil || targetID == "" {
		return "-"
	}
	_, baseline, _, err := s.Repo.GetGucBaseline(ctx)
	if err != nil || len(baseline) == 0 {
		return "-"
	}
	live, _, _, err := s.Repo.GetServerGucSnapshot(ctx, targetID)
	if err != nil || len(live) == 0 {
		return "-"
	}
	n := 0
	for _, row := range postgresconfig.CompareAgainstBaseline(baseline, live) {
		if row.Status != postgresconfig.DriftMatch {
			n++
		}
	}
	if n == 0 {
		return "0"
	}
	return fmt.Sprintf("%d", n)
}

func gucSnapshotHostLabel(s reportstore.GucSnapshotSummary) string {
	host := strings.TrimSpace(s.TargetHost)
	if host == "" {
		return s.TargetID
	}
	return host
}

// GucBaseline returns the stored global baseline.
func (s *Service) GucBaseline(ctx context.Context) (*GucBaselineResponse, error) {
	if s.Repo == nil {
		return &GucBaselineResponse{Settings: map[string]string{}}, nil
	}
	label, settings, updatedAt, err := s.Repo.GetGucBaseline(ctx)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		settings = map[string]string{}
	}
	return &GucBaselineResponse{
		Label:      label,
		Settings:   settings,
		UpdatedAt:  updatedAt,
		KeyCount:   len(settings),
	}, nil
}

// PutGucBaseline upserts the global baseline settings blob.
func (s *Service) PutGucBaseline(ctx context.Context, label string, settings map[string]string) error {
	if s.Repo == nil {
		return fmt.Errorf("database not configured")
	}
	return s.Repo.UpsertGucBaseline(ctx, label, settings)
}

// GucSnapshots lists latest SHOW ALL snapshots per server.
func (s *Service) GucSnapshots(ctx context.Context) (*GucSnapshotsResponse, error) {
	if s.Repo == nil {
		return &GucSnapshotsResponse{Snapshots: []GucSnapshotEntry{}}, nil
	}
	list, err := s.Repo.ListServerGucSnapshots(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]GucSnapshotEntry, 0, len(list))
	for _, s := range list {
		out = append(out, GucSnapshotEntry{
			TargetID:    s.TargetID,
			Host:        gucSnapshotHostLabel(s),
			NodeID:      s.NodeID,
			CollectedAt: s.CollectedAt,
			KeyCount:    s.KeyCount,
		})
	}
	return &GucSnapshotsResponse{Snapshots: out}, nil
}
