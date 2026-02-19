package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropics/three-body-engine/internal/domain"
)

// validJSON returns a minimal valid configuration JSON string.
func validJSON() string {
	return `{
		"db_path": "/tmp/test.db",
		"workspace": "/tmp/workspace",
		"budget_cap_usd": 10.0,
		"providers": {
			"test-provider": {
				"command": "echo",
				"args": ["hello"]
			}
		}
	}`
}

func writeConfig(t *testing.T, dir, content string) string {
	t.Helper()
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return p
}

func TestLoad_Valid(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, validJSON())

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DBPath != "/tmp/test.db" {
		t.Errorf("DBPath = %q, want /tmp/test.db", cfg.DBPath)
	}
	if cfg.Workspace != "/tmp/workspace" {
		t.Errorf("Workspace = %q, want /tmp/workspace", cfg.Workspace)
	}
	if cfg.BudgetCapUSD != 10.0 {
		t.Errorf("BudgetCapUSD = %f, want 10.0", cfg.BudgetCapUSD)
	}
	if len(cfg.Providers) != 1 {
		t.Errorf("Providers count = %d, want 1", len(cfg.Providers))
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `{not valid json}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLoad_MissingDBPath(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `{
		"workspace": "/tmp/ws",
		"budget_cap_usd": 5.0,
		"providers": {"p": {"command": "echo"}}
	}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing db_path, got nil")
	}
	engineErr, ok := err.(*domain.EngineError)
	if !ok {
		t.Fatalf("expected EngineError, got %T", err)
	}
	if engineErr.Code != domain.ErrConfigInvalid.Code {
		t.Errorf("Code = %d, want %d", engineErr.Code, domain.ErrConfigInvalid.Code)
	}
}

func TestLoad_MissingWorkspace(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `{
		"db_path": "/tmp/test.db",
		"budget_cap_usd": 5.0,
		"providers": {"p": {"command": "echo"}}
	}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing workspace, got nil")
	}
	engineErr, ok := err.(*domain.EngineError)
	if !ok {
		t.Fatalf("expected EngineError, got %T", err)
	}
	if engineErr.Code != domain.ErrConfigInvalid.Code {
		t.Errorf("Code = %d, want %d", engineErr.Code, domain.ErrConfigInvalid.Code)
	}
}

func TestLoad_ZeroBudget(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `{
		"db_path": "/tmp/test.db",
		"workspace": "/tmp/ws",
		"budget_cap_usd": 0,
		"providers": {"p": {"command": "echo"}}
	}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for zero budget, got nil")
	}
	engineErr, ok := err.(*domain.EngineError)
	if !ok {
		t.Fatalf("expected EngineError, got %T", err)
	}
	if engineErr.Code != domain.ErrConfigInvalid.Code {
		t.Errorf("Code = %d, want %d", engineErr.Code, domain.ErrConfigInvalid.Code)
	}
}

func TestLoad_NoProviders(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `{
		"db_path": "/tmp/test.db",
		"workspace": "/tmp/ws",
		"budget_cap_usd": 5.0,
		"providers": {}
	}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty providers, got nil")
	}
	engineErr, ok := err.(*domain.EngineError)
	if !ok {
		t.Fatalf("expected EngineError, got %T", err)
	}
	if engineErr.Code != domain.ErrConfigInvalid.Code {
		t.Errorf("Code = %d, want %d", engineErr.Code, domain.ErrConfigInvalid.Code)
	}
}

func TestLoad_DefaultsApplied(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, validJSON())

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.CheckIntervalSec != 10 {
		t.Errorf("CheckIntervalSec = %d, want 10", cfg.CheckIntervalSec)
	}
	if cfg.MaxConcurrentWorkers != 5 {
		t.Errorf("MaxConcurrentWorkers = %d, want 5", cfg.MaxConcurrentWorkers)
	}
	if cfg.ListenAddr != ":9800" {
		t.Errorf("ListenAddr = %q, want :9800", cfg.ListenAddr)
	}
	if cfg.MaxRounds != 3 {
		t.Errorf("MaxRounds = %d, want 3", cfg.MaxRounds)
	}
	if cfg.RateLimitPerMinute != 60 {
		t.Errorf("RateLimitPerMinute = %d, want 60", cfg.RateLimitPerMinute)
	}
}
