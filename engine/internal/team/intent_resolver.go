package team

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
)

// IntentResolver handles acquiring, releasing, and executing file-level intent locks.
type IntentResolver struct {
	DB         *sql.DB
	IntentRepo *store.IntentRepo
	WorkerRepo *store.WorkerRepo
	AuditRepo  *store.AuditRepo
}

// AcquireLock claims an intent lock on a file within a transaction.
// It verifies no conflicting active intents exist and that the worker owns the target file.
func (r *IntentResolver) AcquireLock(ctx context.Context, intent domain.Intent, leaseDurationSec int) error {
	// All reads happen before BeginTx to avoid SQLite single-conn deadlock.
	active, err := r.IntentRepo.FindActiveByFile(ctx, r.DB, intent.TaskID, intent.TargetFile)
	if err != nil {
		return fmt.Errorf("find active intents: %w", err)
	}
	if len(active) > 0 {
		return domain.ErrIntentConflict
	}

	worker, err := r.WorkerRepo.GetByID(ctx, r.DB, intent.WorkerID)
	if err != nil {
		return fmt.Errorf("get worker: %w", err)
	}

	if !ownsFile(worker.FileOwnership, intent.TargetFile) {
		return domain.ErrFileOwnership
	}

	intent.Status = "pending"
	intent.LeaseUntil = time.Now().Unix() + int64(leaseDurationSec)

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.IntentRepo.UpsertTx(ctx, tx, intent); err != nil {
		return fmt.Errorf("upsert intent: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	now := time.Now()
	_ = r.AuditRepo.Record(ctx, r.DB, domain.AuditRecord{
		ID:        fmt.Sprintf("aud-%d", now.UnixNano()),
		TaskID:    intent.TaskID,
		Category:  "intent",
		Actor:     intent.WorkerID,
		Action:    "lock_acquired",
		Severity:  "info",
		CreatedAt: now.Unix(),
	})

	return nil
}

// ReleaseLock cancels an existing intent lock.
func (r *IntentResolver) ReleaseLock(ctx context.Context, intentID string) error {
	// Read before tx to avoid deadlock.
	existing, err := r.IntentRepo.GetByID(ctx, r.DB, intentID)
	if err != nil {
		return err
	}

	existing.Status = "cancelled"

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.IntentRepo.UpsertTx(ctx, tx, *existing); err != nil {
		return fmt.Errorf("upsert cancelled intent: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	now := time.Now()
	_ = r.AuditRepo.Record(ctx, r.DB, domain.AuditRecord{
		ID:        fmt.Sprintf("aud-%d", now.UnixNano()),
		TaskID:    existing.TaskID,
		Category:  "intent",
		Actor:     existing.WorkerID,
		Action:    "lock_released",
		Severity:  "info",
		CreatedAt: now.Unix(),
	})

	return nil
}

// Execute completes an intent by verifying the lease and pre-hash, then marking it done.
func (r *IntentResolver) Execute(ctx context.Context, intentID, currentHash, postHash string) error {
	// Read before tx to avoid deadlock.
	existing, err := r.IntentRepo.GetByID(ctx, r.DB, intentID)
	if err != nil {
		return err
	}

	if existing.LeaseUntil < time.Now().Unix() {
		return domain.ErrLeaseExpired
	}

	if existing.PreHash != currentHash {
		return domain.ErrIntentHashMismatch
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.IntentRepo.MarkDoneTx(ctx, tx, intentID, postHash); err != nil {
		return fmt.Errorf("mark done: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	now := time.Now()
	_ = r.AuditRepo.Record(ctx, r.DB, domain.AuditRecord{
		ID:        fmt.Sprintf("aud-%d", now.UnixNano()),
		TaskID:    existing.TaskID,
		Category:  "intent",
		Actor:     existing.WorkerID,
		Action:    "intent_executed",
		Severity:  "info",
		CreatedAt: now.Unix(),
	})

	return nil
}

func ownsFile(ownership []string, target string) bool {
	for _, f := range ownership {
		if f == target {
			return true
		}
	}
	return false
}
