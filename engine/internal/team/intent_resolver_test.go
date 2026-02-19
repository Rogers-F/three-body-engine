package team

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
)

func newResolverTestDB(t *testing.T) (*IntentResolver, *WorkerManager) {
	t.Helper()
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	mgr := NewWorkerManager(db, 10)
	resolver := &IntentResolver{
		DB:         db,
		IntentRepo: &store.IntentRepo{},
		WorkerRepo: &store.WorkerRepo{},
		AuditRepo:  &store.AuditRepo{},
	}
	return resolver, mgr
}

func spawnTestWorker(t *testing.T, mgr *WorkerManager, files []string) *domain.WorkerRef {
	t.Helper()
	ctx := context.Background()
	w, err := mgr.Spawn(ctx, domain.WorkerSpec{
		TaskID:         "task-1",
		Phase:          domain.PhaseC,
		Role:           "coder",
		FileOwnership:  files,
		SoftTimeoutSec: 300,
		HardTimeoutSec: 600,
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	return w
}

func TestAcquireLock_Success(t *testing.T) {
	resolver, mgr := newResolverTestDB(t)
	ctx := context.Background()
	w := spawnTestWorker(t, mgr, []string{"main.go"})

	intent := domain.Intent{
		IntentID:   "int-1",
		TaskID:     "task-1",
		WorkerID:   w.WorkerID,
		TargetFile: "main.go",
		Operation:  "write",
		PreHash:    "abc",
	}

	if err := resolver.AcquireLock(ctx, intent, 60); err != nil {
		t.Fatalf("AcquireLock: %v", err)
	}

	got, err := resolver.IntentRepo.GetByID(ctx, resolver.DB, "int-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != "pending" {
		t.Errorf("Status = %q, want %q", got.Status, "pending")
	}
	if got.LeaseUntil <= time.Now().Unix() {
		t.Error("LeaseUntil should be in the future")
	}
}

func TestAcquireLock_ConflictingIntent(t *testing.T) {
	resolver, mgr := newResolverTestDB(t)
	ctx := context.Background()
	w := spawnTestWorker(t, mgr, []string{"main.go"})

	intent1 := domain.Intent{
		IntentID:   "int-1",
		TaskID:     "task-1",
		WorkerID:   w.WorkerID,
		TargetFile: "main.go",
		Operation:  "write",
	}
	if err := resolver.AcquireLock(ctx, intent1, 60); err != nil {
		t.Fatalf("AcquireLock first: %v", err)
	}

	intent2 := domain.Intent{
		IntentID:   "int-2",
		TaskID:     "task-1",
		WorkerID:   w.WorkerID,
		TargetFile: "main.go",
		Operation:  "write",
	}
	err := resolver.AcquireLock(ctx, intent2, 60)
	if err != domain.ErrIntentConflict {
		t.Errorf("expected ErrIntentConflict, got %v", err)
	}
}

func TestAcquireLock_FileOwnershipViolation(t *testing.T) {
	resolver, mgr := newResolverTestDB(t)
	ctx := context.Background()
	w := spawnTestWorker(t, mgr, []string{"other.go"})

	intent := domain.Intent{
		IntentID:   "int-1",
		TaskID:     "task-1",
		WorkerID:   w.WorkerID,
		TargetFile: "main.go",
		Operation:  "write",
	}
	err := resolver.AcquireLock(ctx, intent, 60)
	if err != domain.ErrFileOwnership {
		t.Errorf("expected ErrFileOwnership, got %v", err)
	}
}

func TestReleaseLock_Success(t *testing.T) {
	resolver, mgr := newResolverTestDB(t)
	ctx := context.Background()
	w := spawnTestWorker(t, mgr, []string{"main.go"})

	intent := domain.Intent{
		IntentID:   "int-1",
		TaskID:     "task-1",
		WorkerID:   w.WorkerID,
		TargetFile: "main.go",
		Operation:  "write",
	}
	if err := resolver.AcquireLock(ctx, intent, 60); err != nil {
		t.Fatalf("AcquireLock: %v", err)
	}

	if err := resolver.ReleaseLock(ctx, "int-1"); err != nil {
		t.Fatalf("ReleaseLock: %v", err)
	}

	got, err := resolver.IntentRepo.GetByID(ctx, resolver.DB, "int-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != "cancelled" {
		t.Errorf("Status = %q, want %q", got.Status, "cancelled")
	}
}

func TestReleaseLock_NotFound(t *testing.T) {
	resolver, _ := newResolverTestDB(t)
	ctx := context.Background()

	err := resolver.ReleaseLock(ctx, "nonexistent")
	if err != domain.ErrIntentNotFound {
		t.Errorf("expected ErrIntentNotFound, got %v", err)
	}
}

func TestExecute_Success(t *testing.T) {
	resolver, mgr := newResolverTestDB(t)
	ctx := context.Background()
	w := spawnTestWorker(t, mgr, []string{"main.go"})

	intent := domain.Intent{
		IntentID:   "int-1",
		TaskID:     "task-1",
		WorkerID:   w.WorkerID,
		TargetFile: "main.go",
		Operation:  "write",
		PreHash:    "hash-before",
	}
	if err := resolver.AcquireLock(ctx, intent, 120); err != nil {
		t.Fatalf("AcquireLock: %v", err)
	}

	if err := resolver.Execute(ctx, "int-1", "hash-before", "hash-after"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	got, err := resolver.IntentRepo.GetByID(ctx, resolver.DB, "int-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != "done" {
		t.Errorf("Status = %q, want %q", got.Status, "done")
	}
	if got.PostHash != "hash-after" {
		t.Errorf("PostHash = %q, want %q", got.PostHash, "hash-after")
	}
}

func TestExecute_LeaseExpired(t *testing.T) {
	resolver, mgr := newResolverTestDB(t)
	ctx := context.Background()
	w := spawnTestWorker(t, mgr, []string{"main.go"})

	intent := domain.Intent{
		IntentID:   "int-1",
		TaskID:     "task-1",
		WorkerID:   w.WorkerID,
		TargetFile: "main.go",
		Operation:  "write",
		PreHash:    "hash-before",
	}
	// Use a lease of 0 seconds so it expires immediately.
	if err := resolver.AcquireLock(ctx, intent, 0); err != nil {
		t.Fatalf("AcquireLock: %v", err)
	}

	// Wait a moment to ensure expiry.
	time.Sleep(1100 * time.Millisecond)

	err := resolver.Execute(ctx, "int-1", "hash-before", "hash-after")
	if err != domain.ErrLeaseExpired {
		t.Errorf("expected ErrLeaseExpired, got %v", err)
	}
}

func TestExecute_HashMismatch(t *testing.T) {
	resolver, mgr := newResolverTestDB(t)
	ctx := context.Background()
	w := spawnTestWorker(t, mgr, []string{"main.go"})

	intent := domain.Intent{
		IntentID:   "int-1",
		TaskID:     "task-1",
		WorkerID:   w.WorkerID,
		TargetFile: "main.go",
		Operation:  "write",
		PreHash:    "original-hash",
	}
	if err := resolver.AcquireLock(ctx, intent, 120); err != nil {
		t.Fatalf("AcquireLock: %v", err)
	}

	err := resolver.Execute(ctx, "int-1", "different-hash", "hash-after")
	if err != domain.ErrIntentHashMismatch {
		t.Errorf("expected ErrIntentHashMismatch, got %v", err)
	}
}
