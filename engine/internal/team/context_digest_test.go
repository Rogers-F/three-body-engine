package team

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
)

func TestDigestBuilder_Build(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	now := time.Now().Unix()

	// Create a task
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	taskRepo := &store.TaskRepo{}
	err = taskRepo.CreateTx(ctx, tx, domain.FlowState{
		TaskID:        "task-1",
		CurrentPhase:  domain.PhaseC,
		Status:        domain.StatusRunning,
		StateVersion:  1,
		BudgetUsedUSD: 1.5,
		BudgetCapUSD:  10.0,
		UpdatedAtUnix: now,
	})
	if err != nil {
		t.Fatalf("CreateTx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Create a snapshot
	tx2, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	snapRepo := &store.SnapshotRepo{}
	err = snapRepo.SaveTx(ctx, tx2, domain.PhaseSnapshot{
		TaskID:       "task-1",
		Phase:        domain.PhaseC,
		Round:        2,
		SnapshotJSON: `{"data":"test"}`,
		Checksum:     "abc123",
		CreatedAt:    now,
	})
	if err != nil {
		t.Fatalf("SaveTx: %v", err)
	}
	if err := tx2.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Create a pending intent
	tx3, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	intentRepo := &store.IntentRepo{}
	err = intentRepo.UpsertTx(ctx, tx3, domain.Intent{
		IntentID:   "int-1",
		TaskID:     "task-1",
		WorkerID:   "w-1",
		TargetFile: "main.go",
		Operation:  "write",
		Status:     "pending",
	})
	if err != nil {
		t.Fatalf("UpsertTx: %v", err)
	}
	if err := tx3.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	builder := NewDigestBuilder(db)
	spec := domain.WorkerSpec{
		TaskID:         "task-1",
		Phase:          domain.PhaseC,
		Role:           "coder",
		FileOwnership:  []string{"main.go"},
		SoftTimeoutSec: 300,
		HardTimeoutSec: 600,
	}

	digest, err := builder.Build(ctx, "task-1", domain.PhaseC, spec)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if digest.TaskID != "task-1" {
		t.Errorf("TaskID = %q, want %q", digest.TaskID, "task-1")
	}
	if digest.PhaseID != "C" {
		t.Errorf("PhaseID = %q, want %q", digest.PhaseID, "C")
	}
	if len(digest.FileOwnership) != 1 || digest.FileOwnership[0] != "main.go" {
		t.Errorf("FileOwnership = %v, want [main.go]", digest.FileOwnership)
	}
	if len(digest.Constraints) == 0 {
		t.Error("expected non-empty Constraints")
	}
	if len(digest.ArtifactRefs) != 1 {
		t.Errorf("expected 1 artifact ref, got %d", len(digest.ArtifactRefs))
	}
	if digest.ArtifactRefs[0].Path != "main.go" {
		t.Errorf("ArtifactRef path = %q, want %q", digest.ArtifactRefs[0].Path, "main.go")
	}
}

func TestDigestBuilder_MissingSnapshot(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	now := time.Now().Unix()

	// Create task without snapshot
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	taskRepo := &store.TaskRepo{}
	err = taskRepo.CreateTx(ctx, tx, domain.FlowState{
		TaskID:       "task-2",
		CurrentPhase: domain.PhaseA,
		Status:       domain.StatusRunning,
		StateVersion: 1,
		UpdatedAtUnix: now,
	})
	if err != nil {
		t.Fatalf("CreateTx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	builder := NewDigestBuilder(db)
	spec := domain.WorkerSpec{
		TaskID: "task-2",
		Phase:  domain.PhaseA,
		Role:   "explorer",
	}

	digest, err := builder.Build(ctx, "task-2", domain.PhaseA, spec)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if digest.TaskID != "task-2" {
		t.Errorf("TaskID = %q, want %q", digest.TaskID, "task-2")
	}
	if len(digest.ArtifactRefs) != 0 {
		t.Errorf("expected 0 artifact refs with no intents, got %d", len(digest.ArtifactRefs))
	}
}

func TestDigestBuilder_PendingIntentsAsRefs(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	now := time.Now().Unix()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	taskRepo := &store.TaskRepo{}
	err = taskRepo.CreateTx(ctx, tx, domain.FlowState{
		TaskID:       "task-3",
		CurrentPhase: domain.PhaseC,
		Status:       domain.StatusRunning,
		StateVersion: 1,
		UpdatedAtUnix: now,
	})
	if err != nil {
		t.Fatalf("CreateTx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	intents := []domain.Intent{
		{IntentID: "int-a", TaskID: "task-3", TargetFile: "a.go", Operation: "write", Status: "pending"},
		{IntentID: "int-b", TaskID: "task-3", TargetFile: "b.go", Operation: "create", Status: "pending"},
		{IntentID: "int-c", TaskID: "task-3", TargetFile: "c.go", Operation: "write", Status: "done"},
	}
	for _, intent := range intents {
		tx2, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("BeginTx: %v", err)
		}
		intentRepo := &store.IntentRepo{}
		if err := intentRepo.UpsertTx(ctx, tx2, intent); err != nil {
			t.Fatalf("UpsertTx: %v", err)
		}
		if err := tx2.Commit(); err != nil {
			t.Fatalf("Commit: %v", err)
		}
	}

	builder := NewDigestBuilder(db)
	spec := domain.WorkerSpec{TaskID: "task-3", Phase: domain.PhaseC, Role: "coder"}

	digest, err := builder.Build(ctx, "task-3", domain.PhaseC, spec)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// Only pending intents should appear (int-a, int-b, not int-c)
	if len(digest.ArtifactRefs) != 2 {
		t.Fatalf("expected 2 artifact refs for pending intents, got %d", len(digest.ArtifactRefs))
	}
	if digest.ArtifactRefs[0].Path != "a.go" {
		t.Errorf("first ref path = %q, want %q", digest.ArtifactRefs[0].Path, "a.go")
	}
	if digest.ArtifactRefs[1].Path != "b.go" {
		t.Errorf("second ref path = %q, want %q", digest.ArtifactRefs[1].Path, "b.go")
	}
}
