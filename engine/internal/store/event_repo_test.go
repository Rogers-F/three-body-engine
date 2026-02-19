package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/anthropics/three-body-engine/internal/domain"
)

func TestEventRepo_AppendAndList(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &EventRepo{}
	now := time.Now().Unix()

	events := []domain.WorkflowEvent{
		{TaskID: "task-1", SeqNo: 1, Phase: domain.PhaseA, EventType: "phase_start", PayloadJSON: "{}", CreatedAt: now},
		{TaskID: "task-1", SeqNo: 2, Phase: domain.PhaseA, EventType: "phase_end", PayloadJSON: "{}", CreatedAt: now + 1},
		{TaskID: "task-1", SeqNo: 3, Phase: domain.PhaseB, EventType: "phase_start", PayloadJSON: "{}", CreatedAt: now + 2},
	}

	for _, e := range events {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		if err := repo.AppendTx(ctx, tx, e); err != nil {
			t.Fatalf("AppendTx seq=%d: %v", e.SeqNo, err)
		}
		tx.Commit()
	}

	// List all events since seq 0.
	got, err := repo.ListByTask(ctx, db, "task-1", 0)
	if err != nil {
		t.Fatalf("ListByTask: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 events, got %d", len(got))
	}

	// List events since seq 1 (should return seq 2, 3).
	got, err = repo.ListByTask(ctx, db, "task-1", 1)
	if err != nil {
		t.Fatalf("ListByTask sinceSeq=1: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(got))
	}
	if got[0].SeqNo != 2 {
		t.Errorf("first event SeqNo = %d, want 2", got[0].SeqNo)
	}
}

func TestEventRepo_DuplicateSeqNo(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &EventRepo{}
	now := time.Now().Unix()

	event := domain.WorkflowEvent{
		TaskID: "task-dup", SeqNo: 1, Phase: domain.PhaseA,
		EventType: "test", PayloadJSON: "{}", CreatedAt: now,
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := repo.AppendTx(ctx, tx, event); err != nil {
		t.Fatalf("first AppendTx: %v", err)
	}
	tx.Commit()

	// Duplicate (task_id, seq_no) should fail.
	tx2, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	err = repo.AppendTx(ctx, tx2, event)
	tx2.Rollback()

	if err == nil {
		t.Error("expected error on duplicate seq_no, got nil")
	}
}

func TestEventRepo_ListByTask_Empty(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &EventRepo{}

	got, err := repo.ListByTask(ctx, db, "nonexistent", 0)
	if err != nil {
		t.Fatalf("ListByTask: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil slice for empty result, got %v", got)
	}
}
