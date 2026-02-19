package store

import (
	"path/filepath"
	"testing"
)

func TestNewDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	// Verify tables were created by querying sqlite_master.
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		t.Fatalf("query tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		tables = append(tables, name)
	}

	expected := map[string]bool{
		"tasks":           true,
		"workflow_events": true,
		"phase_snapshots": true,
		"audit_records":   true,
		"intent_logs":     true,
	}

	for _, tbl := range tables {
		delete(expected, tbl)
	}
	for tbl := range expected {
		t.Errorf("expected table %q not found", tbl)
	}
}

func TestNewDB_IdempotentMigration(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// First open creates schema.
	db1, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("first NewDB: %v", err)
	}
	db1.Close()

	// Second open should not fail (IF NOT EXISTS).
	db2, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("second NewDB: %v", err)
	}
	db2.Close()
}
