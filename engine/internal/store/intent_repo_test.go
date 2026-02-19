package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/anthropics/three-body-engine/internal/domain"
)

func TestIntentRepo_UpsertAndList(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &IntentRepo{}

	intent := domain.Intent{
		IntentID:    "int-1",
		TaskID:      "task-1",
		WorkerID:    "w-1",
		TargetFile:  "src/main.go",
		Operation:   "write",
		Status:      "pending",
		PreHash:     "aaa",
		PayloadHash: "bbb",
		LeaseUntil:  9999999999,
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := repo.UpsertTx(ctx, tx, intent); err != nil {
		t.Fatalf("UpsertTx: %v", err)
	}
	tx.Commit()

	got, err := repo.ListByTaskStatus(ctx, db, "task-1", "pending")
	if err != nil {
		t.Fatalf("ListByTaskStatus: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 intent, got %d", len(got))
	}
	if got[0].IntentID != "int-1" {
		t.Errorf("IntentID = %q, want %q", got[0].IntentID, "int-1")
	}
	if got[0].TargetFile != "src/main.go" {
		t.Errorf("TargetFile = %q, want %q", got[0].TargetFile, "src/main.go")
	}
}

func TestIntentRepo_UpsertUpdatesExisting(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &IntentRepo{}

	intent := domain.Intent{
		IntentID:   "int-2",
		TaskID:     "task-1",
		WorkerID:   "w-1",
		TargetFile: "old.go",
		Operation:  "write",
		Status:     "pending",
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	repo.UpsertTx(ctx, tx, intent)
	tx.Commit()

	// Upsert with changed target file.
	intent.TargetFile = "new.go"
	tx2, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := repo.UpsertTx(ctx, tx2, intent); err != nil {
		t.Fatalf("UpsertTx update: %v", err)
	}
	tx2.Commit()

	got, err := repo.ListByTaskStatus(ctx, db, "task-1", "pending")
	if err != nil {
		t.Fatalf("ListByTaskStatus: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 intent after upsert, got %d", len(got))
	}
	if got[0].TargetFile != "new.go" {
		t.Errorf("TargetFile = %q, want %q", got[0].TargetFile, "new.go")
	}
}

func TestIntentRepo_MarkDone(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &IntentRepo{}

	intent := domain.Intent{
		IntentID:   "int-3",
		TaskID:     "task-1",
		WorkerID:   "w-1",
		TargetFile: "file.go",
		Operation:  "write",
		Status:     "pending",
		PreHash:    "before",
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	repo.UpsertTx(ctx, tx, intent)
	tx.Commit()

	tx2, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := repo.MarkDoneTx(ctx, tx2, "int-3", "after-hash"); err != nil {
		t.Fatalf("MarkDoneTx: %v", err)
	}
	tx2.Commit()

	// Should no longer appear in pending list.
	pending, err := repo.ListByTaskStatus(ctx, db, "task-1", "pending")
	if err != nil {
		t.Fatalf("ListByTaskStatus pending: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("expected 0 pending intents, got %d", len(pending))
	}

	// Should appear in done list.
	done, err := repo.ListByTaskStatus(ctx, db, "task-1", "done")
	if err != nil {
		t.Fatalf("ListByTaskStatus done: %v", err)
	}
	if len(done) != 1 {
		t.Fatalf("expected 1 done intent, got %d", len(done))
	}
	if done[0].PostHash != "after-hash" {
		t.Errorf("PostHash = %q, want %q", done[0].PostHash, "after-hash")
	}
}

func TestIntentRepo_MarkDone_NotFound(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &IntentRepo{}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	err = repo.MarkDoneTx(ctx, tx, "nonexistent", "hash")
	tx.Rollback()

	if err != domain.ErrIntentNotFound {
		t.Errorf("expected ErrIntentNotFound, got %v", err)
	}
}
