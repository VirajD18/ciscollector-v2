package service

import (
	"context"
	"time"

	"github.com/klouddb/klouddbshield/pkg/repository"
)

// CollectorHeartbeat upserts fleet status from a collector heartbeat.
func (s *Service) CollectorHeartbeat(ctx context.Context, nodeID, hostname, ip string, ts time.Time, cronRunning bool, scheduledJobs int, lastError string) error {
	if err := s.requireRepo(); err != nil {
		return err
	}
	return s.Repo.UpsertCollectorStatus(ctx, nodeID, hostname, ip, ts, cronRunning, scheduledJobs, lastError)
}

// RegisterCollector registers a collector node and records activity.
func (s *Service) RegisterCollector(ctx context.Context, nodeID, hostname, ip string, ts time.Time, scheduledJobs int, activityMsg string) error {
	if err := s.requireRepo(); err != nil {
		return err
	}
	if err := s.Repo.UpsertCollectorStatus(ctx, nodeID, hostname, ip, ts, false, scheduledJobs, ""); err != nil {
		return err
	}
	return s.Repo.InsertCollectorActivity(ctx, nodeID, "register", activityMsg, "info", ts)
}

// RecordCollectorActivity inserts an activity event and updates last seen.
func (s *Service) RecordCollectorActivity(ctx context.Context, nodeID, kind, message, level string, ts time.Time) error {
	if err := s.requireRepo(); err != nil {
		return err
	}
	if err := s.Repo.InsertCollectorActivity(ctx, nodeID, kind, message, level, ts); err != nil {
		return err
	}
	return s.Repo.TouchCollectorLastSeen(ctx, nodeID, ts)
}

// RecordCollectorLog inserts a log line and updates last seen.
func (s *Service) RecordCollectorLog(ctx context.Context, nodeID, level, message string, ts time.Time) error {
	if err := s.requireRepo(); err != nil {
		return err
	}
	if err := s.Repo.InsertCollectorLog(ctx, nodeID, level, message, ts); err != nil {
		return err
	}
	return s.Repo.TouchCollectorLastSeen(ctx, nodeID, ts)
}

// RecordCollectorRun stores a collector run and updates node status.
func (s *Service) RecordCollectorRun(ctx context.Context, nodeID, hostname, trigger string, startedAt, finishedAt time.Time, features []string, success bool, errMsg string) error {
	if err := s.requireRepo(); err != nil {
		return err
	}
	return s.Repo.InsertCollectorRun(ctx, nodeID, hostname, trigger, startedAt, finishedAt, features, success, errMsg)
}

// ListCollectorNodes returns all collector fleet nodes.
func (s *Service) ListCollectorNodes(ctx context.Context) ([]repository.CollectorNodeRow, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	return s.Repo.ListCollectorNodes(ctx)
}

// GetCollectorNode returns one collector node by id.
func (s *Service) GetCollectorNode(ctx context.Context, nodeID string) (*repository.CollectorNodeRow, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	return s.Repo.GetCollectorNode(ctx, nodeID)
}

// ListCollectorRuns returns recent runs for a node.
func (s *Service) ListCollectorRuns(ctx context.Context, nodeID string, limit int) ([]repository.CollectorRunRow, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	return s.Repo.ListCollectorRuns(ctx, nodeID, limit)
}

// ListCollectorActivity returns recent activity for a node.
func (s *Service) ListCollectorActivity(ctx context.Context, nodeID string, limit int) ([]repository.CollectorActivityRow, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	return s.Repo.ListCollectorActivity(ctx, nodeID, limit)
}

// ListCollectorLogs returns recent logs for a node.
func (s *Service) ListCollectorLogs(ctx context.Context, nodeID string, limit int) ([]repository.CollectorLogRow, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	return s.Repo.ListCollectorLogs(ctx, nodeID, limit)
}
