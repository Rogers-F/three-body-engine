package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/anthropics/three-body-engine/internal/domain"
)

func TestSnapshotRepo_SaveAndGetLatest(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &SnapshotRepo{}
	now := time.Now().Unix()

	// Save two snapshots for the same phase.
	snap1 := domain.PhaseSnapshot{
		TaskID: "task-1", Phase: domain.PhaseA, Round: 1,
		SnapshotJSON: `{"round":1}`, Checksum: "abc", CreatedAt: now,
	}
	snap2 := domain.PhaseSnapshot{
		TaskID: "task-1", Phase: domain.PhaseA, Round: 2,
		SnapshotJSON: `{"round":2}`, Checksum: "def", CreatedAt: now + 1,
	}

	for _, s := range []domain.PhaseSnapshot{snap1, snap2} {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		if err := repo.SaveTx(ctx, tx, s); err != nil {
			t.Fatalf("SaveTx round=%d: %v", s.Round, err)
		}
		tx.Commit()
	}

	// GetLatest should return the second snapshot.
	got, err := repo.GetLatest(ctx, db, "task-1", domain.PhaseA)
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if got == nil {
		t.Fatal("expected snapshot, got nil")
	}
	if got.Round != 2 {
		t.Errorf("Round = %d, want 2", got.Round)
	}
	if got.Checksum != "def" {
		t.Errorf("Checksum = %q, want %q", got.Checksum, "def")
	}
}

func TestSnapshotRepo_GetLatest_NoMatch(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &SnapshotRepo{}

	got, err := repo.GetLatest(ctx, db, "nonexistent", domain.PhaseA)
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for no match, got %+v", got)
	}
}

func TestSnapshotRepo_DifferentPhases(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &SnapshotRepo{}
	now := time.Now().Unix()

	snapA := domain.PhaseSnapshot{
		TaskID: "task-1", Phase: domain.PhaseA, Round: 1,
		SnapshotJSON: `{"phase":"A"}`, Checksum: "a1", CreatedAt: now,
	}
	snapB := domain.PhaseSnapshot{
		TaskID: "task-1", Phase: domain.PhaseB, Round: 1,
		SnapshotJSON: `{"phase":"B"}`, Checksum: "b1", CreatedAt: now,
	}

	for _, s := range []domain.PhaseSnapshot{snapA, snapB} {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		if err := repo.SaveTx(ctx, tx, s); err != nil {
			t.Fatalf("SaveTx: %v", err)
		}
		tx.Commit()
	}

	gotA, err := repo.GetLatest(ctx, db, "task-1", domain.PhaseA)
	if err != nil {
		t.Fatalf("GetLatest A: %v", err)
	}
	if gotA.Checksum != "a1" {
		t.Errorf("phase A checksum = %q, want %q", gotA.Checksum, "a1")
	}

	gotB, err := repo.GetLatest(ctx, db, "task-1", domain.PhaseB)
	if err != nil {
		t.Fatalf("GetLatest B: %v", err)
	}
	if gotB.Checksum != "b1" {
		t.Errorf("phase B checksum = %q, want %q", gotB.Checksum, "b1")
	}
}
