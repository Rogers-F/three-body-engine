package team

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
)

// DigestBuilder constructs lightweight context digests for workers.
type DigestBuilder struct {
	DB           *sql.DB
	TaskRepo     *store.TaskRepo
	SnapshotRepo *store.SnapshotRepo
	IntentRepo   *store.IntentRepo
}

// NewDigestBuilder creates a DigestBuilder with default repos.
func NewDigestBuilder(db *sql.DB) *DigestBuilder {
	return &DigestBuilder{
		DB:           db,
		TaskRepo:     &store.TaskRepo{},
		SnapshotRepo: &store.SnapshotRepo{},
		IntentRepo:   &store.IntentRepo{},
	}
}

// Build constructs a ContextDigest for the given task, phase, and worker spec.
func (b *DigestBuilder) Build(ctx context.Context, taskID string, phase domain.Phase, spec domain.WorkerSpec) (*domain.ContextDigest, error) {
	task, err := b.TaskRepo.GetByID(ctx, b.DB, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	snap, err := b.SnapshotRepo.GetLatest(ctx, b.DB, taskID, phase)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	intents, err := b.IntentRepo.ListByTaskStatus(ctx, b.DB, taskID, "pending")
	if err != nil {
		return nil, fmt.Errorf("list pending intents: %w", err)
	}

	digest := &domain.ContextDigest{
		TaskID:        taskID,
		PhaseID:       string(phase),
		Objective:     fmt.Sprintf("[%s] worker in phase %s", spec.Role, string(phase)),
		FileOwnership: spec.FileOwnership,
		Deadline: domain.Deadline{
			Soft: fmt.Sprintf("%ds", spec.SoftTimeoutSec),
			Hard: fmt.Sprintf("%ds", spec.HardTimeoutSec),
		},
	}

	constraints := []string{
		fmt.Sprintf("budget_used=%.2f", task.BudgetUsedUSD),
		fmt.Sprintf("budget_cap=%.2f", task.BudgetCapUSD),
		fmt.Sprintf("phase=%s", string(task.CurrentPhase)),
	}
	if snap != nil {
		constraints = append(constraints, fmt.Sprintf("snapshot_round=%d", snap.Round))
	}
	digest.Constraints = constraints

	var refs []domain.ArtifactRef
	for i, intent := range intents {
		refs = append(refs, domain.ArtifactRef{
			ID:   intent.IntentID,
			Type: intent.Operation,
			Path: intent.TargetFile,
			Version: i + 1,
		})
	}
	digest.ArtifactRefs = refs

	return digest, nil
}
