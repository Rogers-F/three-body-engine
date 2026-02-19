package workflow

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
)

func TestBudgetGovernor_CheckBudget(t *testing.T) {
	tests := []struct {
		name     string
		used     float64
		cap      float64
		expected domain.CostAction
	}{
		{"well_under_budget", 1.0, 10.0, domain.CostContinue},
		{"at_79_percent", 7.9, 10.0, domain.CostContinue},
		{"at_80_percent_warn", 8.0, 10.0, domain.CostWarn},
		{"at_90_percent_warn", 9.0, 10.0, domain.CostWarn},
		{"at_100_percent_halt", 10.0, 10.0, domain.CostHalt},
		{"over_budget_halt", 12.0, 10.0, domain.CostHalt},
		{"zero_cap_continue", 5.0, 0.0, domain.CostContinue},
		{"zero_used_continue", 0.0, 10.0, domain.CostContinue},
	}

	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	gov := NewBudgetGovernor(db)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := domain.FlowState{
				BudgetUsedUSD: tt.used,
				BudgetCapUSD:  tt.cap,
			}
			got, err := gov.CheckBudget(context.Background(), state)
			if err != nil {
				t.Fatalf("CheckBudget: %v", err)
			}
			if got != tt.expected {
				t.Errorf("CheckBudget(used=%f, cap=%f) = %q, want %q", tt.used, tt.cap, got, tt.expected)
			}
		})
	}
}

func TestBudgetGovernor_RecordUsage(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	taskRepo := &store.TaskRepo{}

	// Create a task with a $10 budget.
	state := domain.FlowState{
		TaskID:        "task-budget",
		CurrentPhase:  domain.PhaseA,
		Status:        domain.StatusRunning,
		StateVersion:  1,
		BudgetCapUSD:  10.0,
		BudgetUsedUSD: 0.0,
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	taskRepo.CreateTx(ctx, tx, state)
	tx.Commit()

	gov := NewBudgetGovernor(db)

	// Small usage should return continue.
	action, err := gov.RecordUsage(ctx, "task-budget", domain.CostDelta{AmountUSD: 2.0})
	if err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}
	if action != domain.CostContinue {
		t.Errorf("action = %q, want continue", action)
	}

	// More usage should push past warn threshold.
	action, err = gov.RecordUsage(ctx, "task-budget", domain.CostDelta{AmountUSD: 6.5})
	if err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}
	if action != domain.CostWarn {
		t.Errorf("action = %q, want warn", action)
	}

	// Push past halt threshold.
	action, err = gov.RecordUsage(ctx, "task-budget", domain.CostDelta{AmountUSD: 2.0})
	if err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}
	if action != domain.CostHalt {
		t.Errorf("action = %q, want halt", action)
	}
}

func TestBudgetGovernor_RecordUsage_NotFound(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	gov := NewBudgetGovernor(db)

	_, err = gov.RecordUsage(context.Background(), "nonexistent", domain.CostDelta{AmountUSD: 1.0})
	if err == nil {
		t.Error("expected error for nonexistent task, got nil")
	}
}

func TestBudgetGovernor_CustomThresholds(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	gov := NewBudgetGovernor(db)
	gov.WarnRatio = 0.5
	gov.HaltRatio = 0.9

	state := domain.FlowState{
		BudgetUsedUSD: 5.0,
		BudgetCapUSD:  10.0,
	}

	action, err := gov.CheckBudget(context.Background(), state)
	if err != nil {
		t.Fatalf("CheckBudget: %v", err)
	}
	if action != domain.CostWarn {
		t.Errorf("action = %q at 50%% with 50%% threshold, want warn", action)
	}
}
