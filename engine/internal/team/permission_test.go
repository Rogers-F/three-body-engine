package team

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
)

func TestPermissionBroker_BuildCapabilitySheet(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	broker := NewPermissionBroker(db)
	sheet := broker.BuildCapabilitySheet("task-1",
		[]string{"src/", "tests/"},
		[]string{"read", "write"},
	)

	if sheet.TaskID != "task-1" {
		t.Errorf("TaskID = %q, want %q", sheet.TaskID, "task-1")
	}
	if len(sheet.AllowedPaths) != 2 {
		t.Errorf("AllowedPaths len = %d, want 2", len(sheet.AllowedPaths))
	}
	if len(sheet.AllowedCommands) != 2 {
		t.Errorf("AllowedCommands len = %d, want 2", len(sheet.AllowedCommands))
	}
	if len(sheet.DeniedPatterns) == 0 {
		t.Error("expected default denied patterns")
	}
	if sheet.CreatedAtUnix == 0 {
		t.Error("expected non-zero CreatedAtUnix")
	}
}

func TestPermissionBroker_AllowsValidPathAndCommand(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	broker := NewPermissionBroker(db)
	sheet := &domain.CapabilitySheet{
		TaskID:          "task-1",
		AllowedPaths:    []string{"src/"},
		AllowedCommands: []string{"read", "write"},
		DeniedPatterns:  defaultDeniedPatterns,
	}

	allowed, err := broker.CheckPermission(context.Background(), sheet, "src/main.go", "read")
	if err != nil {
		t.Fatalf("CheckPermission: %v", err)
	}
	if !allowed {
		t.Error("expected permission to be allowed")
	}
}

func TestPermissionBroker_DeniesPathNotInAllowed(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	broker := NewPermissionBroker(db)
	sheet := &domain.CapabilitySheet{
		TaskID:          "task-1",
		AllowedPaths:    []string{"src/"},
		AllowedCommands: []string{"read"},
		DeniedPatterns:  defaultDeniedPatterns,
	}

	allowed, err := broker.CheckPermission(context.Background(), sheet, "secret/data.txt", "read")
	if err != nil {
		t.Fatalf("CheckPermission: %v", err)
	}
	if allowed {
		t.Error("expected permission to be denied for path not in allowed list")
	}
}

func TestPermissionBroker_DeniesCommandNotInAllowed(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	broker := NewPermissionBroker(db)
	sheet := &domain.CapabilitySheet{
		TaskID:          "task-1",
		AllowedPaths:    []string{"src/"},
		AllowedCommands: []string{"read"},
		DeniedPatterns:  defaultDeniedPatterns,
	}

	allowed, err := broker.CheckPermission(context.Background(), sheet, "src/main.go", "delete")
	if err != nil {
		t.Fatalf("CheckPermission: %v", err)
	}
	if allowed {
		t.Error("expected permission to be denied for command not in allowed list")
	}
}

func TestPermissionBroker_DeniesEnvPaths(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	broker := NewPermissionBroker(db)
	sheet := &domain.CapabilitySheet{
		TaskID:          "task-1",
		AllowedPaths:    []string{"./"},
		AllowedCommands: []string{"read"},
		DeniedPatterns:  defaultDeniedPatterns,
	}

	allowed, err := broker.CheckPermission(context.Background(), sheet, ".env", "read")
	if err != nil {
		t.Fatalf("CheckPermission: %v", err)
	}
	if allowed {
		t.Error("expected .env to be denied")
	}
}

func TestPermissionBroker_AuditsDenials(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	broker := NewPermissionBroker(db)
	sheet := &domain.CapabilitySheet{
		TaskID:          "task-1",
		AllowedPaths:    []string{"src/"},
		AllowedCommands: []string{"read"},
		DeniedPatterns:  defaultDeniedPatterns,
	}

	_, _ = broker.CheckPermission(context.Background(), sheet, "forbidden/file.go", "read")

	auditRepo := &store.AuditRepo{}
	records, err := auditRepo.ListByTask(context.Background(), db, "task-1")
	if err != nil {
		t.Fatalf("ListByTask: %v", err)
	}
	if len(records) == 0 {
		t.Error("expected audit record for denied permission")
	}
	found := false
	for _, r := range records {
		if r.Action == "permission_denied" && r.Severity == "warning" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected audit record with action=permission_denied and severity=warning")
	}
}
