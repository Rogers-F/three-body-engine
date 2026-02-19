package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/anthropics/three-body-engine/internal/domain"
)

// IntentRepo handles persistence for Intent records.
type IntentRepo struct{}

// UpsertTx inserts or updates an intent within an existing transaction.
func (r *IntentRepo) UpsertTx(ctx context.Context, tx *sql.Tx, intent domain.Intent) error {
	const q = `INSERT INTO intent_logs (intent_id, task_id, worker_id, target_file, operation, status, pre_hash, post_hash, payload_hash, lease_until)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(intent_id) DO UPDATE SET
	worker_id = excluded.worker_id,
	target_file = excluded.target_file,
	operation = excluded.operation,
	status = excluded.status,
	pre_hash = excluded.pre_hash,
	post_hash = excluded.post_hash,
	payload_hash = excluded.payload_hash,
	lease_until = excluded.lease_until`

	_, err := tx.ExecContext(ctx, q,
		intent.IntentID,
		intent.TaskID,
		intent.WorkerID,
		intent.TargetFile,
		intent.Operation,
		intent.Status,
		intent.PreHash,
		intent.PostHash,
		intent.PayloadHash,
		intent.LeaseUntil,
	)
	if err != nil {
		return fmt.Errorf("upsert intent: %w", err)
	}
	return nil
}

// ListByTaskStatus returns intents for a task filtered by status.
func (r *IntentRepo) ListByTaskStatus(ctx context.Context, db *sql.DB, taskID, status string) ([]domain.Intent, error) {
	const q = `SELECT intent_id, task_id, worker_id, target_file, operation, status, pre_hash, post_hash, payload_hash, lease_until
FROM intent_logs
WHERE task_id = ? AND status = ?
ORDER BY intent_id ASC`

	rows, err := db.QueryContext(ctx, q, taskID, status)
	if err != nil {
		return nil, fmt.Errorf("list intents: %w", err)
	}
	defer rows.Close()

	var intents []domain.Intent
	for rows.Next() {
		var i domain.Intent
		if err := rows.Scan(&i.IntentID, &i.TaskID, &i.WorkerID, &i.TargetFile, &i.Operation,
			&i.Status, &i.PreHash, &i.PostHash, &i.PayloadHash, &i.LeaseUntil); err != nil {
			return nil, fmt.Errorf("scan intent: %w", err)
		}
		intents = append(intents, i)
	}
	return intents, rows.Err()
}

// MarkDoneTx marks an intent as done with a post-operation hash within a transaction.
func (r *IntentRepo) MarkDoneTx(ctx context.Context, tx *sql.Tx, intentID, postHash string) error {
	const q = `UPDATE intent_logs SET status = 'done', post_hash = ? WHERE intent_id = ?`
	res, err := tx.ExecContext(ctx, q, postHash, intentID)
	if err != nil {
		return fmt.Errorf("mark intent done: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrIntentNotFound
	}
	return nil
}
