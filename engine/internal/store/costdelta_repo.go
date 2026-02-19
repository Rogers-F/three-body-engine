package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/anthropics/three-body-engine/internal/domain"
)

// CostDeltaRepo handles persistence for CostDelta records.
type CostDeltaRepo struct{}

// Create inserts a new cost delta record for a task.
func (r *CostDeltaRepo) Create(ctx context.Context, db *sql.DB, taskID string, delta domain.CostDelta) error {
	const q = `INSERT INTO cost_deltas (task_id, input_tokens, output_tokens, amount_usd, provider, phase, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := db.ExecContext(ctx, q,
		taskID,
		delta.InputTokens,
		delta.OutputTokens,
		delta.AmountUSD,
		string(delta.Provider),
		string(delta.Phase),
		delta.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create cost delta: %w", err)
	}
	return nil
}

// ListByTask returns all cost deltas for a task, ordered by creation time.
func (r *CostDeltaRepo) ListByTask(ctx context.Context, db *sql.DB, taskID string) ([]domain.CostDelta, error) {
	const q = `SELECT input_tokens, output_tokens, amount_usd, provider, phase, created_at
FROM cost_deltas
WHERE task_id = ?
ORDER BY created_at ASC`

	rows, err := db.QueryContext(ctx, q, taskID)
	if err != nil {
		return nil, fmt.Errorf("list cost deltas: %w", err)
	}
	defer rows.Close()

	var deltas []domain.CostDelta
	for rows.Next() {
		var d domain.CostDelta
		var provider, phase string
		if err := rows.Scan(&d.InputTokens, &d.OutputTokens, &d.AmountUSD, &provider, &phase, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan cost delta: %w", err)
		}
		d.Provider = domain.Provider(provider)
		d.Phase = domain.Phase(phase)
		deltas = append(deltas, d)
	}
	return deltas, rows.Err()
}
