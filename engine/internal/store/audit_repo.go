package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/anthropics/three-body-engine/internal/domain"
)

// AuditRepo handles persistence for AuditRecord entries.
type AuditRepo struct{}

// Record inserts an audit record.
func (r *AuditRepo) Record(ctx context.Context, db *sql.DB, rec domain.AuditRecord) error {
	const q = `INSERT INTO audit_records (id, task_id, category, actor, action, request_json, decision_json, severity, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := db.ExecContext(ctx, q,
		rec.ID,
		rec.TaskID,
		rec.Category,
		rec.Actor,
		rec.Action,
		rec.RequestJSON,
		rec.DecisionJSON,
		rec.Severity,
		rec.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("record audit: %w", err)
	}
	return nil
}

// ListByTask returns all audit records for a given task, ordered by creation time.
func (r *AuditRepo) ListByTask(ctx context.Context, db *sql.DB, taskID string) ([]domain.AuditRecord, error) {
	const q = `SELECT id, task_id, category, actor, action, request_json, decision_json, severity, created_at
FROM audit_records
WHERE task_id = ?
ORDER BY created_at ASC`

	rows, err := db.QueryContext(ctx, q, taskID)
	if err != nil {
		return nil, fmt.Errorf("list audit records: %w", err)
	}
	defer rows.Close()

	var records []domain.AuditRecord
	for rows.Next() {
		var a domain.AuditRecord
		if err := rows.Scan(&a.ID, &a.TaskID, &a.Category, &a.Actor, &a.Action,
			&a.RequestJSON, &a.DecisionJSON, &a.Severity, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit record: %w", err)
		}
		records = append(records, a)
	}
	return records, rows.Err()
}
