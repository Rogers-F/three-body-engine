package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/anthropics/three-body-engine/internal/domain"
)

// ScoreCardRepo handles persistence for ScoreCard records.
type ScoreCardRepo struct{}

// Create inserts a new score card record.
func (r *ScoreCardRepo) Create(ctx context.Context, db *sql.DB, card domain.ScoreCard) error {
	issuesJSON, err := json.Marshal(card.Issues)
	if err != nil {
		return fmt.Errorf("marshal issues: %w", err)
	}
	altsJSON, err := json.Marshal(card.Alternatives)
	if err != nil {
		return fmt.Errorf("marshal alternatives: %w", err)
	}

	const q = `INSERT INTO score_cards (review_id, task_id, reviewer, correctness, security, maintainability, cost, delivery_risk, issues_json, alternatives_json, verdict, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = db.ExecContext(ctx, q,
		card.ReviewID,
		card.TaskID,
		card.Reviewer,
		card.Scores.Correctness,
		card.Scores.Security,
		card.Scores.Maintainability,
		card.Scores.Cost,
		card.Scores.DeliveryRisk,
		string(issuesJSON),
		string(altsJSON),
		card.Verdict,
		card.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create score card: %w", err)
	}
	return nil
}

// ListByTask returns all score cards for a task, ordered by creation time.
func (r *ScoreCardRepo) ListByTask(ctx context.Context, db *sql.DB, taskID string) ([]domain.ScoreCard, error) {
	const q = `SELECT review_id, task_id, reviewer, correctness, security, maintainability, cost, delivery_risk, issues_json, alternatives_json, verdict, created_at
FROM score_cards
WHERE task_id = ?
ORDER BY created_at ASC`

	rows, err := db.QueryContext(ctx, q, taskID)
	if err != nil {
		return nil, fmt.Errorf("list score cards: %w", err)
	}
	defer rows.Close()

	var cards []domain.ScoreCard
	for rows.Next() {
		var c domain.ScoreCard
		var issuesJSON, altsJSON string
		if err := rows.Scan(
			&c.ReviewID, &c.TaskID, &c.Reviewer,
			&c.Scores.Correctness, &c.Scores.Security, &c.Scores.Maintainability,
			&c.Scores.Cost, &c.Scores.DeliveryRisk,
			&issuesJSON, &altsJSON,
			&c.Verdict, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan score card: %w", err)
		}
		if err := json.Unmarshal([]byte(issuesJSON), &c.Issues); err != nil {
			return nil, fmt.Errorf("unmarshal issues: %w", err)
		}
		if err := json.Unmarshal([]byte(altsJSON), &c.Alternatives); err != nil {
			return nil, fmt.Errorf("unmarshal alternatives: %w", err)
		}
		cards = append(cards, c)
	}
	return cards, rows.Err()
}
