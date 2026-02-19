package team

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
)

var workerSeq atomic.Int64

// WorkerManager handles spawning, replacing, and shutting down workers.
type WorkerManager struct {
	DB         *sql.DB
	WorkerRepo *store.WorkerRepo
	AuditRepo  *store.AuditRepo
	MaxWorkers int
}

// NewWorkerManager creates a WorkerManager with the given database and max worker limit.
func NewWorkerManager(db *sql.DB, maxWorkers int) *WorkerManager {
	return &WorkerManager{
		DB:         db,
		WorkerRepo: &store.WorkerRepo{},
		AuditRepo:  &store.AuditRepo{},
		MaxWorkers: maxWorkers,
	}
}

// Spawn creates a new worker from the given spec, enforcing the max worker limit.
func (m *WorkerManager) Spawn(ctx context.Context, spec domain.WorkerSpec) (*domain.WorkerRef, error) {
	count, err := m.WorkerRepo.CountActive(ctx, m.DB, spec.TaskID)
	if err != nil {
		return nil, fmt.Errorf("count active workers: %w", err)
	}
	if count >= m.MaxWorkers {
		return nil, domain.ErrWorkerLimitReached
	}

	now := time.Now()
	seq := workerSeq.Add(1)

	ownership := spec.FileOwnership
	if ownership == nil {
		ownership = []string{}
	}

	w := domain.WorkerRef{
		WorkerID:       fmt.Sprintf("w-%d-%d", now.UnixNano(), seq),
		TaskID:         spec.TaskID,
		Phase:          spec.Phase,
		Role:           spec.Role,
		State:          domain.WorkerCreated,
		FileOwnership:  ownership,
		SoftTimeoutSec: spec.SoftTimeoutSec,
		HardTimeoutSec: spec.HardTimeoutSec,
		LastHeartbeat:  now.Unix(),
		CreatedAtUnix:  now.Unix(),
	}

	if err := m.WorkerRepo.Create(ctx, m.DB, w); err != nil {
		return nil, fmt.Errorf("create worker: %w", err)
	}

	_ = m.AuditRepo.Record(ctx, m.DB, domain.AuditRecord{
		ID:        fmt.Sprintf("aud-%d", now.UnixNano()),
		TaskID:    spec.TaskID,
		Category:  "worker",
		Actor:     "system",
		Action:    "worker_spawned",
		Severity:  "info",
		CreatedAt: now.Unix(),
	})

	return &w, nil
}

// UpdateState changes a worker's state, preventing transitions from terminal states.
func (m *WorkerManager) UpdateState(ctx context.Context, workerID string, state domain.WorkerState) error {
	existing, err := m.WorkerRepo.GetByID(ctx, m.DB, workerID)
	if err != nil {
		return err
	}

	if isTerminal(existing.State) {
		return domain.ErrWorkerAlreadyDone
	}

	return m.WorkerRepo.UpdateState(ctx, m.DB, workerID, state)
}

// Replace marks an existing worker as replaced and spawns a new one with the same spec.
func (m *WorkerManager) Replace(ctx context.Context, workerID string) (*domain.WorkerRef, error) {
	old, err := m.WorkerRepo.GetByID(ctx, m.DB, workerID)
	if err != nil {
		return nil, err
	}

	if err := m.WorkerRepo.UpdateState(ctx, m.DB, workerID, domain.WorkerReplaced); err != nil {
		return nil, fmt.Errorf("mark worker as replaced: %w", err)
	}

	spec := domain.WorkerSpec{
		TaskID:         old.TaskID,
		Phase:          old.Phase,
		Role:           old.Role,
		FileOwnership:  old.FileOwnership,
		SoftTimeoutSec: old.SoftTimeoutSec,
		HardTimeoutSec: old.HardTimeoutSec,
	}

	return m.Spawn(ctx, spec)
}

// Shutdown marks a worker as done and records an audit event.
func (m *WorkerManager) Shutdown(ctx context.Context, workerID string) error {
	existing, err := m.WorkerRepo.GetByID(ctx, m.DB, workerID)
	if err != nil {
		return err
	}

	if err := m.WorkerRepo.UpdateState(ctx, m.DB, workerID, domain.WorkerDone); err != nil {
		return fmt.Errorf("shutdown worker: %w", err)
	}

	now := time.Now()
	_ = m.AuditRepo.Record(ctx, m.DB, domain.AuditRecord{
		ID:        fmt.Sprintf("aud-%d", now.UnixNano()),
		TaskID:    existing.TaskID,
		Category:  "worker",
		Actor:     "system",
		Action:    "worker_shutdown",
		Severity:  "info",
		CreatedAt: now.Unix(),
	})

	return nil
}

// ListActive returns all active workers for a task.
func (m *WorkerManager) ListActive(ctx context.Context, taskID string) ([]*domain.WorkerRef, error) {
	return m.WorkerRepo.ListActive(ctx, m.DB, taskID)
}

func isTerminal(s domain.WorkerState) bool {
	return s == domain.WorkerDone || s == domain.WorkerReplaced || s == domain.WorkerHardTimeout
}
