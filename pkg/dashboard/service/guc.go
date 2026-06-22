package service

import (
	"context"
)

// GucDrift returns fleet-wide GUC drift from baseline vs SHOW ALL snapshots.
func (s *Service) GucDrift(ctx context.Context) (*GucDriftResponse, error) {
	return s.gucDriftFromSnapshots(ctx)
}
