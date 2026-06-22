package service

import (
	"context"
	"time"
)

// Overview builds fleet overview from latest runs per target.
func (s *Service) Overview(ctx context.Context) (*OverviewResponse, error) {
	runs, err := s.latestRunsByTarget(ctx)
	if err != nil {
		return nil, err
	}
	resp := &OverviewResponse{
		UpdatedAt: time.Now().UTC(),
		Servers:   []OverviewServer{},
	}
	healthy, warning, critical := 0, 0, 0
	for _, run := range runs {
		cis := decodeCISResults(run.Report)
		passN, failN, score := runCISSummary(run)
		if len(cis) > 0 && passN+failN == 0 {
			passN, failN, score = summarizeCISResults(cis)
		}
		critN := countCriticalFails(cis)
		if critN == 0 {
			critN = failN
		}
		status := hostStatus(score, failN)
		switch status {
		case "Passing":
			healthy++
		case "Failing":
			if critN > 0 {
				critical++
			} else {
				warning++
			}
		default:
			warning++
		}
		srv := OverviewServer{
			ID:     run.TargetID,
			Name:   hostLabel(run),
			IP:     run.TargetHost,
			Status: status,
		}
		srv.ServerSummary.TotalCases = passN + failN
		srv.ServerSummary.PassedCases = passN
		resp.Servers = append(resp.Servers, srv)
	}
	resp.Summary.Servers = len(resp.Servers)
	resp.Summary.Healthy = healthy
	resp.Summary.Warning = warning
	resp.Summary.Critical = critical
	if len(resp.Servers) > 0 {
		resp.CentralServer = resp.Servers[0]
	} else {
		resp.CentralServer = OverviewServer{ID: "local", Name: "No scans yet", Status: "Unknown"}
	}
	return resp, nil
}

