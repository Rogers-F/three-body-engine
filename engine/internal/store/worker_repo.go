package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/anthropics/three-body-engine/internal/domain"
)

// WorkerRepo handles persistence for WorkerRef records.
type WorkerRepo struct{}

// Create inserts a new worker record.
func (r *WorkerRepo) Create(ctx context.Context, db *sql.DB, w domain.WorkerRef) error {
	ownership, err := json.Marshal(w.FileOwnership)
	if err != nil {
		return fmt.Errorf("marshal file_ownership: %w", err)
	}

	const q = `INSERT INTO workers (worker_id, task_id, phase, role, state, file_ownership, soft_timeout_sec, hard_timeout_sec, last_heartbeat, created_at_unix)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = db.ExecContext(ctx, q,
		w.WorkerID,
		w.TaskID,
		string(w.Phase),
		w.Role,
		string(w.State),
		string(ownership),
		w.SoftTimeoutSec,
		w.HardTimeoutSec,
		w.LastHeartbeat,
		w.CreatedAtUnix,
	)
	if err != nil {
		return fmt.Errorf("create worker: %w", err)
	}
	return nil
}

// UpdateState changes the state of a worker by ID.
func (r *WorkerRepo) UpdateState(ctx context.Context, db *sql.DB, workerID string, state domain.WorkerState) error {
	const q = `UPDATE workers SET state = ? WHERE worker_id = ?`
	res, err := db.ExecContext(ctx, q, string(state), workerID)
	if err != nil {
		return fmt.Errorf("update worker state: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrWorkerNotFound
	}
	return nil
}

// GetByID retrieves a worker by its ID.
func (r *WorkerRepo) GetByID(ctx context.Context, db *sql.DB, workerID string) (*domain.WorkerRef, error) {
	const q = `SELECT worker_id, task_id, phase, role, state, file_ownership, soft_timeout_sec, hard_timeout_sec, last_heartbeat, created_at_unix
FROM workers WHERE worker_id = ?`

	row := db.QueryRowContext(ctx, q, workerID)

	var w domain.WorkerRef
	var phase, state, ownershipJSON string
	err := row.Scan(&w.WorkerID, &w.TaskID, &phase, &w.Role, &state, &ownershipJSON,
		&w.SoftTimeoutSec, &w.HardTimeoutSec, &w.LastHeartbeat, &w.CreatedAtUnix)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrWorkerNotFound
		}
		return nil, fmt.Errorf("get worker: %w", err)
	}
	w.Phase = domain.Phase(phase)
	w.State = domain.WorkerState(state)

	if err := json.Unmarshal([]byte(ownershipJSON), &w.FileOwnership); err != nil {
		return nil, fmt.Errorf("unmarshal file_ownership: %w", err)
	}
	return &w, nil
}

// ListActive returns workers for a task that are in created or running state.
func (r *WorkerRepo) ListActive(ctx context.Context, db *sql.DB, taskID string) ([]*domain.WorkerRef, error) {
	const q = `SELECT worker_id, task_id, phase, role, state, file_ownership, soft_timeout_sec, hard_timeout_sec, last_heartbeat, created_at_unix
FROM workers WHERE task_id = ? AND state IN ('created', 'running')
ORDER BY created_at_unix ASC`

	rows, err := db.QueryContext(ctx, q, taskID)
	if err != nil {
		return nil, fmt.Errorf("list active workers: %w", err)
	}
	defer rows.Close()

	var workers []*domain.WorkerRef
	for rows.Next() {
		var w domain.WorkerRef
		var phase, state, ownershipJSON string
		if err := rows.Scan(&w.WorkerID, &w.TaskID, &phase, &w.Role, &state, &ownershipJSON,
			&w.SoftTimeoutSec, &w.HardTimeoutSec, &w.LastHeartbeat, &w.CreatedAtUnix); err != nil {
			return nil, fmt.Errorf("scan worker: %w", err)
		}
		w.Phase = domain.Phase(phase)
		w.State = domain.WorkerState(state)
		if err := json.Unmarshal([]byte(ownershipJSON), &w.FileOwnership); err != nil {
			return nil, fmt.Errorf("unmarshal file_ownership: %w", err)
		}
		workers = append(workers, &w)
	}
	return workers, rows.Err()
}

// ListByTask returns all workers for a task regardless of state, ordered by creation time.
func (r *WorkerRepo) ListByTask(ctx context.Context, db *sql.DB, taskID string) ([]*domain.WorkerRef, error) {
	const q = `SELECT worker_id, task_id, phase, role, state, file_ownership, soft_timeout_sec, hard_timeout_sec, last_heartbeat, created_at_unix
FROM workers WHERE task_id = ?
ORDER BY created_at_unix ASC`

	rows, err := db.QueryContext(ctx, q, taskID)
	if err != nil {
		return nil, fmt.Errorf("list workers by task: %w", err)
	}
	defer rows.Close()

	var workers []*domain.WorkerRef
	for rows.Next() {
		var w domain.WorkerRef
		var phase, state, ownershipJSON string
		if err := rows.Scan(&w.WorkerID, &w.TaskID, &phase, &w.Role, &state, &ownershipJSON,
			&w.SoftTimeoutSec, &w.HardTimeoutSec, &w.LastHeartbeat, &w.CreatedAtUnix); err != nil {
			return nil, fmt.Errorf("scan worker: %w", err)
		}
		w.Phase = domain.Phase(phase)
		w.State = domain.WorkerState(state)
		if err := json.Unmarshal([]byte(ownershipJSON), &w.FileOwnership); err != nil {
			return nil, fmt.Errorf("unmarshal file_ownership: %w", err)
		}
		workers = append(workers, &w)
	}
	return workers, rows.Err()
}

// UpdateHeartbeat updates the last_heartbeat timestamp for a worker.
func (r *WorkerRepo) UpdateHeartbeat(ctx context.Context, db *sql.DB, workerID string, ts int64) error {
	const q = `UPDATE workers SET last_heartbeat = ? WHERE worker_id = ?`
	res, err := db.ExecContext(ctx, q, ts, workerID)
	if err != nil {
		return fmt.Errorf("update heartbeat: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrWorkerNotFound
	}
	return nil
}

// CountActive returns the number of active (created or running) workers for a task.
func (r *WorkerRepo) CountActive(ctx context.Context, db *sql.DB, taskID string) (int, error) {
	const q = `SELECT COUNT(*) FROM workers WHERE task_id = ? AND state IN ('created', 'running')`
	var count int
	err := db.QueryRowContext(ctx, q, taskID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active workers: %w", err)
	}
	return count, nil
}
