package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/anthropics/three-body-engine/internal/domain"
)

// EventRepo handles persistence for WorkflowEvent records.
type EventRepo struct{}

// AppendTx inserts a workflow event within an existing transaction.
func (r *EventRepo) AppendTx(ctx context.Context, tx *sql.Tx, event domain.WorkflowEvent) error {
	const q = `INSERT INTO workflow_events (task_id, seq_no, phase, event_type, payload_json, created_at)
VALUES (?, ?, ?, ?, ?, ?)`
	_, err := tx.ExecContext(ctx, q,
		event.TaskID,
		event.SeqNo,
		string(event.Phase),
		event.EventType,
		event.PayloadJSON,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("append event: %w", err)
	}
	return nil
}

// ListByTask returns events for a task with sequence numbers greater than sinceSeq,
// ordered by sequence number ascending.
func (r *EventRepo) ListByTask(ctx context.Context, db *sql.DB, taskID string, sinceSeq int64) ([]domain.WorkflowEvent, error) {
	const q = `SELECT id, task_id, seq_no, phase, event_type, payload_json, created_at
FROM workflow_events
WHERE task_id = ? AND seq_no > ?
ORDER BY seq_no ASC`

	rows, err := db.QueryContext(ctx, q, taskID, sinceSeq)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var events []domain.WorkflowEvent
	for rows.Next() {
		var e domain.WorkflowEvent
		var phase string
		if err := rows.Scan(&e.ID, &e.TaskID, &e.SeqNo, &phase, &e.EventType, &e.PayloadJSON, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		e.Phase = domain.Phase(phase)
		events = append(events, e)
	}
	return events, rows.Err()
}
