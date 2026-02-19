package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/anthropics/three-body-engine/internal/domain"
)

func TestTaskRepo_CreateAndGet(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &TaskRepo{}

	state := domain.FlowState{
		TaskID:       "task-001",
		CurrentPhase: domain.PhaseA,
		Status:       domain.StatusRunning,
		StateVersion: 1,
		Round:        0,
		BudgetCapUSD: 10.0,
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if err := repo.CreateTx(ctx, tx, state); err != nil {
		t.Fatalf("CreateTx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	got, err := repo.GetByID(ctx, db, "task-001")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.TaskID != "task-001" {
		t.Errorf("TaskID = %q, want %q", got.TaskID, "task-001")
	}
	if got.CurrentPhase != domain.PhaseA {
		t.Errorf("Phase = %q, want %q", got.CurrentPhase, domain.PhaseA)
	}
	if got.Status != domain.StatusRunning {
		t.Errorf("Status = %q, want %q", got.Status, domain.StatusRunning)
	}
	if got.StateVersion != 1 {
		t.Errorf("StateVersion = %d, want 1", got.StateVersion)
	}
	if got.BudgetCapUSD != 10.0 {
		t.Errorf("BudgetCapUSD = %f, want 10.0", got.BudgetCapUSD)
	}
}

func TestTaskRepo_GetByID_NotFound(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &TaskRepo{}

	_, err = repo.GetByID(ctx, db, "nonexistent")
	if err != domain.ErrFlowNotFound {
		t.Errorf("expected ErrFlowNotFound, got %v", err)
	}
}

func TestTaskRepo_UpdateState_OptimisticLock(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &TaskRepo{}

	state := domain.FlowState{
		TaskID:       "task-002",
		CurrentPhase: domain.PhaseA,
		Status:       domain.StatusRunning,
		StateVersion: 1,
		BudgetCapUSD: 5.0,
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := repo.CreateTx(ctx, tx, state); err != nil {
		t.Fatalf("CreateTx: %v", err)
	}
	tx.Commit()

	// Update with correct version should succeed.
	state.CurrentPhase = domain.PhaseB
	tx2, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := repo.UpdateStateTx(ctx, tx2, state); err != nil {
		t.Fatalf("UpdateStateTx: %v", err)
	}
	tx2.Commit()

	// Update with stale version should fail.
	state.CurrentPhase = domain.PhaseC
	// state.StateVersion is still 1 but DB is now 2
	tx3, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	err = repo.UpdateStateTx(ctx, tx3, state)
	tx3.Rollback()

	if err != domain.ErrOptimisticLock {
		t.Errorf("expected ErrOptimisticLock, got %v", err)
	}
}

func TestTaskRepo_DuplicateCreate(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &TaskRepo{}

	state := domain.FlowState{
		TaskID:       "task-dup",
		CurrentPhase: domain.PhaseA,
		Status:       domain.StatusRunning,
		StateVersion: 1,
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := repo.CreateTx(ctx, tx, state); err != nil {
		t.Fatalf("first CreateTx: %v", err)
	}
	tx.Commit()

	tx2, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	err = repo.CreateTx(ctx, tx2, state)
	tx2.Rollback()

	if err == nil {
		t.Error("expected error on duplicate create, got nil")
	}
}
