package guard

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
	"github.com/anthropics/three-body-engine/internal/team"
	"github.com/anthropics/three-body-engine/internal/workflow"
)

// setupGuard creates a DB, task, and Guard for testing.
func setupGuard(t *testing.T, round int, budgetUsed, budgetCap float64) *Guard {
	t.Helper()
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	taskRepo := &store.TaskRepo{}

	state := domain.FlowState{
		TaskID:        "task-1",
		CurrentPhase:  domain.PhaseA,
		Status:        domain.StatusRunning,
		StateVersion:  1,
		Round:         round,
		BudgetUsedUSD: budgetUsed,
		BudgetCapUSD:  budgetCap,
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := taskRepo.CreateTx(ctx, tx, state); err != nil {
		t.Fatalf("CreateTx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	gov := workflow.NewBudgetGovernor(db)
	broker := team.NewPermissionBroker(db)

	return NewGuard(db, gov, broker, GuardConfig{
		MaxRounds:          3,
		RateLimitPerMinute: 5,
	})
}

func defaultSheet() *domain.CapabilitySheet {
	return &domain.CapabilitySheet{
		TaskID:          "task-1",
		AllowedPaths:    []string{"/workspace/"},
		AllowedCommands: []string{"read", "write"},
		DeniedPatterns:  []string{".env"},
	}
}

func TestCheckAll_PassesClean(t *testing.T) {
	g := setupGuard(t, 0, 1.0, 10.0)
	err := g.CheckAll(context.Background(), "task-1", "/workspace/main.go", "read", defaultSheet())
	if err != nil {
		t.Fatalf("CheckAll should pass: %v", err)
	}
}

func TestCheckAll_BudgetExceeded(t *testing.T) {
	g := setupGuard(t, 0, 10.0, 10.0)
	err := g.CheckAll(context.Background(), "task-1", "/workspace/main.go", "read", defaultSheet())
	if err != domain.ErrBudgetExceeded {
		t.Fatalf("expected ErrBudgetExceeded, got %v", err)
	}
}

func TestCheckAll_PermissionDenied(t *testing.T) {
	g := setupGuard(t, 0, 1.0, 10.0)
	err := g.CheckAll(context.Background(), "task-1", "/forbidden/secret.go", "read", defaultSheet())
	if err != domain.ErrPermissionDenied {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestCheckAll_RateLimitExceeded(t *testing.T) {
	g := setupGuard(t, 0, 1.0, 10.0)
	ctx := context.Background()
	sheet := defaultSheet()

	// Exhaust the rate limit (limit is 5).
	for i := 0; i < 5; i++ {
		if err := g.CheckAll(ctx, "task-1", "/workspace/main.go", "read", sheet); err != nil {
			t.Fatalf("CheckAll iteration %d: %v", i, err)
		}
	}

	// Next call should hit rate limit.
	err := g.CheckAll(ctx, "task-1", "/workspace/main.go", "read", sheet)
	if err != domain.ErrRateLimitExceeded {
		t.Fatalf("expected ErrRateLimitExceeded, got %v", err)
	}
}

func TestCheckAll_MaxRoundsExceeded(t *testing.T) {
	g := setupGuard(t, 3, 1.0, 10.0)
	// Set a high rate limit so it doesn't interfere.
	g.Config.RateLimitPerMinute = 100
	err := g.CheckAll(context.Background(), "task-1", "/workspace/main.go", "read", defaultSheet())
	if err != domain.ErrMaxRoundsExceeded {
		t.Fatalf("expected ErrMaxRoundsExceeded, got %v", err)
	}
}

func TestCheckBudget_Continue(t *testing.T) {
	g := setupGuard(t, 0, 1.0, 10.0)
	action, err := g.CheckBudget(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("CheckBudget: %v", err)
	}
	if action != domain.CostContinue {
		t.Errorf("action = %q, want continue", action)
	}
}

func TestCheckBudget_Warn(t *testing.T) {
	g := setupGuard(t, 0, 8.5, 10.0)
	action, err := g.CheckBudget(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("CheckBudget: %v", err)
	}
	if action != domain.CostWarn {
		t.Errorf("action = %q, want warn", action)
	}
}

func TestCheckBudget_Halt(t *testing.T) {
	g := setupGuard(t, 0, 10.0, 10.0)
	action, err := g.CheckBudget(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("CheckBudget: %v", err)
	}
	if action != domain.CostHalt {
		t.Errorf("action = %q, want halt", action)
	}
}

func TestCheckRateLimit_WithinLimit(t *testing.T) {
	g := setupGuard(t, 0, 1.0, 10.0)
	for i := 0; i < 5; i++ {
		if err := g.CheckRateLimit("task-1"); err != nil {
			t.Fatalf("CheckRateLimit iteration %d: %v", i, err)
		}
	}
}

func TestCheckRateLimit_WindowResets(t *testing.T) {
	g := setupGuard(t, 0, 1.0, 10.0)

	// Fill the bucket up to the limit.
	for i := 0; i < 5; i++ {
		if err := g.CheckRateLimit("task-1"); err != nil {
			t.Fatalf("CheckRateLimit iteration %d: %v", i, err)
		}
	}

	// Should be rate limited now.
	if err := g.CheckRateLimit("task-1"); err != domain.ErrRateLimitExceeded {
		t.Fatalf("expected ErrRateLimitExceeded, got %v", err)
	}

	// Simulate window reset by moving windowStart back.
	g.mu.Lock()
	g.rateCounts["task-1"].windowStart -= 61
	g.mu.Unlock()

	// After window reset, should succeed again.
	if err := g.CheckRateLimit("task-1"); err != nil {
		t.Fatalf("CheckRateLimit after window reset: %v", err)
	}
}
