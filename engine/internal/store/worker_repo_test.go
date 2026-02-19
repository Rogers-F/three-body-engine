package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/anthropics/three-body-engine/internal/domain"
)

func TestWorkerRepo_CreateAndGetByID(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &WorkerRepo{}
	now := time.Now().Unix()

	w := domain.WorkerRef{
		WorkerID:       "w-1",
		TaskID:         "task-1",
		Phase:          domain.PhaseC,
		Role:           "coder",
		State:          domain.WorkerCreated,
		FileOwnership:  []string{"main.go", "util.go"},
		SoftTimeoutSec: 300,
		HardTimeoutSec: 600,
		LastHeartbeat:  now,
		CreatedAtUnix:  now,
	}

	if err := repo.Create(ctx, db, w); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, db, "w-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.WorkerID != w.WorkerID {
		t.Errorf("WorkerID = %q, want %q", got.WorkerID, w.WorkerID)
	}
	if got.TaskID != w.TaskID {
		t.Errorf("TaskID = %q, want %q", got.TaskID, w.TaskID)
	}
	if got.Phase != w.Phase {
		t.Errorf("Phase = %q, want %q", got.Phase, w.Phase)
	}
	if got.Role != w.Role {
		t.Errorf("Role = %q, want %q", got.Role, w.Role)
	}
	if got.State != w.State {
		t.Errorf("State = %q, want %q", got.State, w.State)
	}
	if len(got.FileOwnership) != 2 {
		t.Errorf("FileOwnership len = %d, want 2", len(got.FileOwnership))
	}
	if got.SoftTimeoutSec != w.SoftTimeoutSec {
		t.Errorf("SoftTimeoutSec = %d, want %d", got.SoftTimeoutSec, w.SoftTimeoutSec)
	}
}

func TestWorkerRepo_UpdateState(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &WorkerRepo{}
	now := time.Now().Unix()

	w := domain.WorkerRef{
		WorkerID:       "w-2",
		TaskID:         "task-1",
		Phase:          domain.PhaseC,
		Role:           "coder",
		State:          domain.WorkerCreated,
		FileOwnership:  []string{},
		SoftTimeoutSec: 300,
		HardTimeoutSec: 600,
		LastHeartbeat:  now,
		CreatedAtUnix:  now,
	}

	if err := repo.Create(ctx, db, w); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.UpdateState(ctx, db, "w-2", domain.WorkerRunning); err != nil {
		t.Fatalf("UpdateState: %v", err)
	}

	got, err := repo.GetByID(ctx, db, "w-2")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.State != domain.WorkerRunning {
		t.Errorf("State = %q, want %q", got.State, domain.WorkerRunning)
	}
}

func TestWorkerRepo_ListActive(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &WorkerRepo{}
	now := time.Now().Unix()

	workers := []domain.WorkerRef{
		{WorkerID: "w-a", TaskID: "task-1", Phase: domain.PhaseC, State: domain.WorkerCreated, FileOwnership: []string{}, CreatedAtUnix: now},
		{WorkerID: "w-b", TaskID: "task-1", Phase: domain.PhaseC, State: domain.WorkerRunning, FileOwnership: []string{}, CreatedAtUnix: now + 1},
		{WorkerID: "w-c", TaskID: "task-1", Phase: domain.PhaseC, State: domain.WorkerDone, FileOwnership: []string{}, CreatedAtUnix: now + 2},
		{WorkerID: "w-d", TaskID: "task-2", Phase: domain.PhaseC, State: domain.WorkerCreated, FileOwnership: []string{}, CreatedAtUnix: now + 3},
	}

	for _, w := range workers {
		if err := repo.Create(ctx, db, w); err != nil {
			t.Fatalf("Create %s: %v", w.WorkerID, err)
		}
	}

	active, err := repo.ListActive(ctx, db, "task-1")
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(active) != 2 {
		t.Fatalf("expected 2 active workers for task-1, got %d", len(active))
	}
	if active[0].WorkerID != "w-a" {
		t.Errorf("first active worker = %q, want %q", active[0].WorkerID, "w-a")
	}
	if active[1].WorkerID != "w-b" {
		t.Errorf("second active worker = %q, want %q", active[1].WorkerID, "w-b")
	}
}

func TestWorkerRepo_CountActive(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &WorkerRepo{}
	now := time.Now().Unix()

	workers := []domain.WorkerRef{
		{WorkerID: "w-1", TaskID: "task-1", Phase: domain.PhaseC, State: domain.WorkerCreated, FileOwnership: []string{}, CreatedAtUnix: now},
		{WorkerID: "w-2", TaskID: "task-1", Phase: domain.PhaseC, State: domain.WorkerRunning, FileOwnership: []string{}, CreatedAtUnix: now + 1},
		{WorkerID: "w-3", TaskID: "task-1", Phase: domain.PhaseC, State: domain.WorkerDone, FileOwnership: []string{}, CreatedAtUnix: now + 2},
	}

	for _, w := range workers {
		if err := repo.Create(ctx, db, w); err != nil {
			t.Fatalf("Create %s: %v", w.WorkerID, err)
		}
	}

	count, err := repo.CountActive(ctx, db, "task-1")
	if err != nil {
		t.Fatalf("CountActive: %v", err)
	}
	if count != 2 {
		t.Errorf("CountActive = %d, want 2", count)
	}
}

func TestWorkerRepo_GetByID_NotFound(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &WorkerRepo{}

	_, err = repo.GetByID(ctx, db, "nonexistent")
	if err != domain.ErrWorkerNotFound {
		t.Errorf("expected ErrWorkerNotFound, got %v", err)
	}
}
