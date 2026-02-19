package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/anthropics/three-body-engine/internal/domain"
)

// TaskRepo handles persistence for FlowState records.
type TaskRepo struct{}

// CreateTx inserts a new task within an existing transaction.
func (r *TaskRepo) CreateTx(ctx context.Context, tx *sql.Tx, state domain.FlowState) error {
	const q = `INSERT INTO tasks (task_id, current_phase, status, state_version, round, budget_used_usd, budget_cap_usd, last_event_seq, updated_at_unix)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := tx.ExecContext(ctx, q,
		state.TaskID,
		string(state.CurrentPhase),
		string(state.Status),
		state.StateVersion,
		state.Round,
		state.BudgetUsedUSD,
		state.BudgetCapUSD,
		state.LastEventSeq,
		state.UpdatedAtUnix,
	)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

// UpdateStateTx updates a task within a transaction using optimistic locking.
// The update only succeeds if the current state_version matches the expected version.
func (r *TaskRepo) UpdateStateTx(ctx context.Context, tx *sql.Tx, state domain.FlowState) error {
	const q = `UPDATE tasks SET
		current_phase = ?,
		status = ?,
		state_version = state_version + 1,
		round = ?,
		budget_used_usd = ?,
		budget_cap_usd = ?,
		last_event_seq = ?,
		updated_at_unix = ?
	WHERE task_id = ? AND state_version = ?`

	res, err := tx.ExecContext(ctx, q,
		string(state.CurrentPhase),
		string(state.Status),
		state.Round,
		state.BudgetUsedUSD,
		state.BudgetCapUSD,
		state.LastEventSeq,
		state.UpdatedAtUnix,
		state.TaskID,
		state.StateVersion,
	)
	if err != nil {
		return fmt.Errorf("update task state: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrOptimisticLock
	}
	return nil
}

// GetByID retrieves a task by its ID.
func (r *TaskRepo) GetByID(ctx context.Context, db *sql.DB, taskID string) (*domain.FlowState, error) {
	const q = `SELECT task_id, current_phase, status, state_version, round, budget_used_usd, budget_cap_usd, last_event_seq, updated_at_unix
FROM tasks WHERE task_id = ?`

	row := db.QueryRowContext(ctx, q, taskID)

	var s domain.FlowState
	var phase, status string
	err := row.Scan(&s.TaskID, &phase, &status, &s.StateVersion, &s.Round,
		&s.BudgetUsedUSD, &s.BudgetCapUSD, &s.LastEventSeq, &s.UpdatedAtUnix)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrFlowNotFound
		}
		return nil, fmt.Errorf("get task: %w", err)
	}
	s.CurrentPhase = domain.Phase(phase)
	s.Status = domain.FlowStatus(status)
	return &s, nil
}
