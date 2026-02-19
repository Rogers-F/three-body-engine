package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/anthropics/three-body-engine/internal/domain"
)

// SnapshotRepo handles persistence for PhaseSnapshot records.
type SnapshotRepo struct{}

// SaveTx inserts a phase snapshot within an existing transaction.
func (r *SnapshotRepo) SaveTx(ctx context.Context, tx *sql.Tx, snap domain.PhaseSnapshot) error {
	const q = `INSERT INTO phase_snapshots (task_id, phase, round, snapshot_json, checksum, created_at)
VALUES (?, ?, ?, ?, ?, ?)`
	_, err := tx.ExecContext(ctx, q,
		snap.TaskID,
		string(snap.Phase),
		snap.Round,
		snap.SnapshotJSON,
		snap.Checksum,
		snap.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}
	return nil
}

// GetLatest returns the most recent snapshot for a task and phase.
// Returns nil if no snapshot exists.
func (r *SnapshotRepo) GetLatest(ctx context.Context, db *sql.DB, taskID string, phase domain.Phase) (*domain.PhaseSnapshot, error) {
	const q = `SELECT id, task_id, phase, round, snapshot_json, checksum, created_at
FROM phase_snapshots
WHERE task_id = ? AND phase = ?
ORDER BY created_at DESC
LIMIT 1`

	row := db.QueryRowContext(ctx, q, taskID, string(phase))

	var s domain.PhaseSnapshot
	var p string
	err := row.Scan(&s.ID, &s.TaskID, &p, &s.Round, &s.SnapshotJSON, &s.Checksum, &s.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest snapshot: %w", err)
	}
	s.Phase = domain.Phase(p)
	return &s, nil
}
