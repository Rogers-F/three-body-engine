package team

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
)

func newConflictTestDB(t *testing.T) *ConflictDetector {
	t.Helper()
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	return &ConflictDetector{
		IntentRepo: &store.IntentRepo{},
		DB:         db,
	}
}

func insertIntent(t *testing.T, db *store.IntentRepo, sqlDB interface{ Begin() (*interface{}, error) }, intent domain.Intent) {
	// This is a helper; use the actual sql.DB.
}

func insertTestIntent(t *testing.T, detector *ConflictDetector, intent domain.Intent) {
	t.Helper()
	ctx := context.Background()
	tx, err := detector.DB.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if err := detector.IntentRepo.UpsertTx(ctx, tx, intent); err != nil {
		tx.Rollback()
		t.Fatalf("UpsertTx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func TestDetect_NoConflicts(t *testing.T) {
	detector := newConflictTestDB(t)
	ctx := context.Background()

	insertTestIntent(t, detector, domain.Intent{
		IntentID:   "int-1",
		TaskID:     "task-1",
		WorkerID:   "w-1",
		TargetFile: "a.go",
		Operation:  "write",
		Status:     "pending",
	})
	insertTestIntent(t, detector, domain.Intent{
		IntentID:   "int-2",
		TaskID:     "task-1",
		WorkerID:   "w-2",
		TargetFile: "b.go",
		Operation:  "write",
		Status:     "pending",
	})

	conflicts, err := detector.Detect(ctx, "task-1")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(conflicts))
	}
}

func TestDetect_SingleConflict(t *testing.T) {
	detector := newConflictTestDB(t)
	ctx := context.Background()

	insertTestIntent(t, detector, domain.Intent{
		IntentID:   "int-1",
		TaskID:     "task-1",
		WorkerID:   "w-1",
		TargetFile: "main.go",
		Operation:  "write",
		Status:     "pending",
	})
	insertTestIntent(t, detector, domain.Intent{
		IntentID:   "int-2",
		TaskID:     "task-1",
		WorkerID:   "w-2",
		TargetFile: "main.go",
		Operation:  "write",
		Status:     "running",
	})

	conflicts, err := detector.Detect(ctx, "task-1")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Type != ConflictOverlap {
		t.Errorf("Type = %q, want %q", conflicts[0].Type, ConflictOverlap)
	}
}

func TestDetect_MultipleConflicts(t *testing.T) {
	detector := newConflictTestDB(t)
	ctx := context.Background()

	// Three intents on the same file yields 3 pairs.
	insertTestIntent(t, detector, domain.Intent{
		IntentID:   "int-1",
		TaskID:     "task-1",
		WorkerID:   "w-1",
		TargetFile: "main.go",
		Operation:  "write",
		Status:     "pending",
	})
	insertTestIntent(t, detector, domain.Intent{
		IntentID:   "int-2",
		TaskID:     "task-1",
		WorkerID:   "w-2",
		TargetFile: "main.go",
		Operation:  "write",
		Status:     "pending",
	})
	insertTestIntent(t, detector, domain.Intent{
		IntentID:   "int-3",
		TaskID:     "task-1",
		WorkerID:   "w-3",
		TargetFile: "main.go",
		Operation:  "write",
		Status:     "running",
	})

	conflicts, err := detector.Detect(ctx, "task-1")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(conflicts) != 3 {
		t.Errorf("expected 3 conflicts (3 pairs), got %d", len(conflicts))
	}
}

func TestDetectBetween_Overlap(t *testing.T) {
	detector := newConflictTestDB(t)
	a := domain.Intent{TargetFile: "main.go", Operation: "write"}
	b := domain.Intent{TargetFile: "main.go", Operation: "write"}

	c := detector.DetectBetween(a, b)
	if c == nil {
		t.Fatal("expected conflict, got nil")
	}
	if c.Type != ConflictOverlap {
		t.Errorf("Type = %q, want %q", c.Type, ConflictOverlap)
	}
}

func TestDetectBetween_Delete(t *testing.T) {
	detector := newConflictTestDB(t)
	a := domain.Intent{TargetFile: "main.go", Operation: "write"}
	b := domain.Intent{TargetFile: "main.go", Operation: "delete"}

	c := detector.DetectBetween(a, b)
	if c == nil {
		t.Fatal("expected conflict, got nil")
	}
	if c.Type != ConflictDelete {
		t.Errorf("Type = %q, want %q", c.Type, ConflictDelete)
	}
}

func TestDetectBetween_Create(t *testing.T) {
	detector := newConflictTestDB(t)
	a := domain.Intent{TargetFile: "new.go", Operation: "create"}
	b := domain.Intent{TargetFile: "new.go", Operation: "create"}

	c := detector.DetectBetween(a, b)
	if c == nil {
		t.Fatal("expected conflict, got nil")
	}
	if c.Type != ConflictCreate {
		t.Errorf("Type = %q, want %q", c.Type, ConflictCreate)
	}
}

func TestResolve_AlwaysErrors(t *testing.T) {
	detector := newConflictTestDB(t)
	ctx := context.Background()

	conflict := FileConflict{
		File:    "main.go",
		IntentA: domain.Intent{IntentID: "int-1"},
		IntentB: domain.Intent{IntentID: "int-2"},
		Type:    ConflictOverlap,
	}

	err := detector.Resolve(ctx, conflict)
	if err != domain.ErrIntentConflict {
		t.Errorf("expected ErrIntentConflict, got %v", err)
	}
}
