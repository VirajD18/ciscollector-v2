package reportstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const gucBaselineRowID = "global"

// GucSnapshotSummary is a lightweight view of a stored SHOW ALL snapshot.
type GucSnapshotSummary struct {
	TargetID    string
	TargetHost  string
	NodeID      string
	CollectedAt string
	KeyCount    int
}

// UpsertGucBaseline stores the active global GUC baseline as one JSON blob.
func UpsertGucBaseline(ctx context.Context, db *sql.DB, label string, settings map[string]string) error {
	if settings == nil {
		settings = map[string]string{}
	}
	if label == "" {
		label = "global"
	}
	blob, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal baseline: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.ExecContext(ctx, `
		INSERT INTO guc_baseline (id, label, settings_json, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			label = excluded.label,
			settings_json = excluded.settings_json,
			updated_at = excluded.updated_at
	`, gucBaselineRowID, label, string(blob), now)
	return err
}

// GetGucBaseline returns the active global baseline settings.
func GetGucBaseline(ctx context.Context, db *sql.DB) (label string, settings map[string]string, updatedAt string, err error) {
	var blob string
	err = db.QueryRowContext(ctx, `
		SELECT label, settings_json, updated_at FROM guc_baseline WHERE id = ?
	`, gucBaselineRowID).Scan(&label, &blob, &updatedAt)
	if err == sql.ErrNoRows {
		return "", map[string]string{}, "", nil
	}
	if err != nil {
		return "", nil, "", err
	}
	settings = map[string]string{}
	if blob != "" {
		if err := json.Unmarshal([]byte(blob), &settings); err != nil {
			return "", nil, "", fmt.Errorf("decode baseline: %w", err)
		}
	}
	return label, settings, updatedAt, nil
}

// UpsertServerGucSnapshot stores the latest SHOW ALL snapshot for a target.
func UpsertServerGucSnapshot(ctx context.Context, db *sql.DB, targetID, targetHost, nodeID string, settings map[string]string) error {
	if targetID == "" {
		return fmt.Errorf("target_id is required")
	}
	if settings == nil {
		settings = map[string]string{}
	}
	blob, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	if nodeID == "" {
		nodeID = uuid.NewString()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.ExecContext(ctx, `
		INSERT INTO server_guc_snapshots (target_id, target_host, node_id, settings_json, collected_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(target_id) DO UPDATE SET
			target_host = excluded.target_host,
			node_id = excluded.node_id,
			settings_json = excluded.settings_json,
			collected_at = excluded.collected_at
	`, targetID, targetHost, nodeID, string(blob), now)
	return err
}

// GetServerGucSnapshot returns settings for one target.
func GetServerGucSnapshot(ctx context.Context, db *sql.DB, targetID string) (map[string]string, string, string, error) {
	var blob, host, collectedAt string
	err := db.QueryRowContext(ctx, `
		SELECT target_host, settings_json, collected_at
		FROM server_guc_snapshots WHERE target_id = ?
	`, targetID).Scan(&host, &blob, &collectedAt)
	if err == sql.ErrNoRows {
		return nil, "", "", nil
	}
	if err != nil {
		return nil, "", "", err
	}
	settings := map[string]string{}
	if blob != "" {
		if err := json.Unmarshal([]byte(blob), &settings); err != nil {
			return nil, "", "", fmt.Errorf("decode snapshot: %w", err)
		}
	}
	return settings, host, collectedAt, nil
}

// ListServerGucSnapshots returns all latest snapshots for debugging.
func ListServerGucSnapshots(ctx context.Context, db *sql.DB) ([]GucSnapshotSummary, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT target_id, target_host, node_id, collected_at, settings_json
		FROM server_guc_snapshots
		ORDER BY target_host ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []GucSnapshotSummary
	for rows.Next() {
		var s GucSnapshotSummary
		var blob string
		if err := rows.Scan(&s.TargetID, &s.TargetHost, &s.NodeID, &s.CollectedAt, &blob); err != nil {
			return nil, err
		}
		settings := map[string]string{}
		if blob != "" {
			_ = json.Unmarshal([]byte(blob), &settings)
		}
		s.KeyCount = len(settings)
		out = append(out, s)
	}
	return out, rows.Err()
}
