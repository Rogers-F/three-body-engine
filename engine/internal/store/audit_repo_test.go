package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/anthropics/three-body-engine/internal/domain"
)

func TestAuditRepo_RecordAndList(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &AuditRepo{}
	now := time.Now().Unix()

	records := []domain.AuditRecord{
		{ID: "aud-1", TaskID: "task-1", Category: "security", Actor: "system", Action: "check_permissions", RequestJSON: "{}", DecisionJSON: `{"allowed":true}`, Severity: "info", CreatedAt: now},
		{ID: "aud-2", TaskID: "task-1", Category: "budget", Actor: "governor", Action: "check_budget", RequestJSON: "{}", DecisionJSON: `{"action":"continue"}`, Severity: "info", CreatedAt: now + 1},
		{ID: "aud-3", TaskID: "task-2", Category: "security", Actor: "system", Action: "deny", RequestJSON: "{}", DecisionJSON: `{"allowed":false}`, Severity: "warn", CreatedAt: now + 2},
	}

	for _, r := range records {
		if err := repo.Record(ctx, db, r); err != nil {
			t.Fatalf("Record %s: %v", r.ID, err)
		}
	}

	// List by task-1 should return 2 records.
	got, err := repo.ListByTask(ctx, db, "task-1")
	if err != nil {
		t.Fatalf("ListByTask: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 records, got %d", len(got))
	}
	if got[0].ID != "aud-1" {
		t.Errorf("first record ID = %q, want %q", got[0].ID, "aud-1")
	}
	if got[1].ID != "aud-2" {
		t.Errorf("second record ID = %q, want %q", got[1].ID, "aud-2")
	}
}

func TestAuditRepo_DuplicateID(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &AuditRepo{}

	rec := domain.AuditRecord{
		ID: "aud-dup", TaskID: "task-1", Category: "test",
		Action: "test", CreatedAt: time.Now().Unix(),
	}

	if err := repo.Record(ctx, db, rec); err != nil {
		t.Fatalf("first Record: %v", err)
	}

	err = repo.Record(ctx, db, rec)
	if err == nil {
		t.Error("expected error on duplicate ID, got nil")
	}
}

func TestAuditRepo_ListByTask_Empty(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &AuditRepo{}

	got, err := repo.ListByTask(ctx, db, "nonexistent")
	if err != nil {
		t.Fatalf("ListByTask: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for empty result, got %v", got)
	}
}
