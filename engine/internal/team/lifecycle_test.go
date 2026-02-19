package team

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
)

func testSpec() domain.WorkerSpec {
	return domain.WorkerSpec{
		TaskID:         "task-1",
		Phase:          domain.PhaseC,
		Role:           "coder",
		FileOwnership:  []string{"main.go"},
		SoftTimeoutSec: 300,
		HardTimeoutSec: 600,
	}
}

func TestWorkerManager_Spawn(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	mgr := NewWorkerManager(db, 4)
	ctx := context.Background()

	w, err := mgr.Spawn(ctx, testSpec())
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if w.WorkerID == "" {
		t.Error("expected non-empty WorkerID")
	}
	if w.State != domain.WorkerCreated {
		t.Errorf("State = %q, want %q", w.State, domain.WorkerCreated)
	}
	if w.TaskID != "task-1" {
		t.Errorf("TaskID = %q, want %q", w.TaskID, "task-1")
	}
}

func TestWorkerManager_SpawnRespectsLimit(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	mgr := NewWorkerManager(db, 2)
	ctx := context.Background()
	spec := testSpec()

	if _, err := mgr.Spawn(ctx, spec); err != nil {
		t.Fatalf("Spawn 1: %v", err)
	}
	if _, err := mgr.Spawn(ctx, spec); err != nil {
		t.Fatalf("Spawn 2: %v", err)
	}

	_, err = mgr.Spawn(ctx, spec)
	if err == nil {
		t.Fatal("expected error when exceeding max workers")
	}
	if err != domain.ErrWorkerLimitReached {
		t.Errorf("expected ErrWorkerLimitReached, got %v", err)
	}
}

func TestWorkerManager_Replace(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	mgr := NewWorkerManager(db, 4)
	ctx := context.Background()

	old, err := mgr.Spawn(ctx, testSpec())
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	newW, err := mgr.Replace(ctx, old.WorkerID)
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}

	if newW.WorkerID == old.WorkerID {
		t.Error("new worker should have different ID from old")
	}
	if newW.Role != old.Role {
		t.Errorf("new worker Role = %q, want %q", newW.Role, old.Role)
	}

	// Old worker should be marked as replaced
	oldRef, err := mgr.WorkerRepo.GetByID(ctx, mgr.DB, old.WorkerID)
	if err != nil {
		t.Fatalf("GetByID old: %v", err)
	}
	if oldRef.State != domain.WorkerReplaced {
		t.Errorf("old worker State = %q, want %q", oldRef.State, domain.WorkerReplaced)
	}
}

func TestWorkerManager_Shutdown(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	mgr := NewWorkerManager(db, 4)
	ctx := context.Background()

	w, err := mgr.Spawn(ctx, testSpec())
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	if err := mgr.Shutdown(ctx, w.WorkerID); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	got, err := mgr.WorkerRepo.GetByID(ctx, mgr.DB, w.WorkerID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.State != domain.WorkerDone {
		t.Errorf("State = %q, want %q", got.State, domain.WorkerDone)
	}
}

func TestWorkerManager_UpdateStateTerminal(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	mgr := NewWorkerManager(db, 4)
	ctx := context.Background()

	w, err := mgr.Spawn(ctx, testSpec())
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	if err := mgr.Shutdown(ctx, w.WorkerID); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	err = mgr.UpdateState(ctx, w.WorkerID, domain.WorkerRunning)
	if err != domain.ErrWorkerAlreadyDone {
		t.Errorf("expected ErrWorkerAlreadyDone, got %v", err)
	}
}

func TestWorkerManager_ListActive(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	mgr := NewWorkerManager(db, 4)
	ctx := context.Background()
	spec := testSpec()

	w1, err := mgr.Spawn(ctx, spec)
	if err != nil {
		t.Fatalf("Spawn 1: %v", err)
	}
	if _, err := mgr.Spawn(ctx, spec); err != nil {
		t.Fatalf("Spawn 2: %v", err)
	}

	if err := mgr.Shutdown(ctx, w1.WorkerID); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	active, err := mgr.ListActive(ctx, "task-1")
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(active) != 1 {
		t.Errorf("expected 1 active worker, got %d", len(active))
	}
}
