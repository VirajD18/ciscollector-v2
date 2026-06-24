package service

import (
	"context"
)

// Runs lists recent persisted scan runs.
func (s *Service) Runs(ctx context.Context, limit int) (*RunsResponse, error) {
	rows, err := s.Repo.GetRuns(ctx, limit)
	if err != nil {
		return nil, err
	}
	resp := &RunsResponse{Runs: make([]RunSummary, 0, len(rows))}
	for _, r := range rows {
		resp.Runs = append(resp.Runs, RunSummary{
			ID:           r.ID,
			StartedAt:    r.StartedAt,
			Trigger:      r.Trigger,
			TargetID:     r.TargetID,
			TargetHost:   r.TargetHost,
			OverallScore: r.OverallScore,
			TotalPass:    r.TotalPass,
			TotalFail:    r.TotalFail,
		})
	}
	return resp, nil
}
