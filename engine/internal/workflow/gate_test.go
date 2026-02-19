package workflow

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
)

func TestDefaultGate_AllowsRunning(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	gov := NewBudgetGovernor(db)
	gate := &DefaultGate{Governor: gov}

	state := domain.FlowState{
		TaskID:       "task-1",
		Status:       domain.StatusRunning,
		BudgetCapUSD: 10.0,
		BudgetUsedUSD: 2.0,
	}

	decision, err := gate.Evaluate(context.Background(), state)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !decision.Allow {
		t.Errorf("expected Allow=true, got false; blockers: %v", decision.Blockers)
	}
}

func TestDefaultGate_BlocksNonRunning(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	gov := NewBudgetGovernor(db)
	gate := &DefaultGate{Governor: gov}

	tests := []struct {
		name   string
		status domain.FlowStatus
	}{
		{"blocked", domain.StatusBlocked},
		{"failed", domain.StatusFailed},
		{"done", domain.StatusDone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := domain.FlowState{
				TaskID:       "task-1",
				Status:       tt.status,
				BudgetCapUSD: 10.0,
			}
			decision, err := gate.Evaluate(context.Background(), state)
			if err != nil {
				t.Fatalf("Evaluate: %v", err)
			}
			if decision.Allow {
				t.Error("expected Allow=false for non-running status")
			}
			if len(decision.Blockers) == 0 {
				t.Error("expected at least one blocker")
			}
		})
	}
}

func TestDefaultGate_BlocksOverBudget(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	gov := NewBudgetGovernor(db)
	gate := &DefaultGate{Governor: gov}

	state := domain.FlowState{
		TaskID:        "task-1",
		Status:        domain.StatusRunning,
		BudgetCapUSD:  10.0,
		BudgetUsedUSD: 10.0, // At 100% -> halt
	}

	decision, err := gate.Evaluate(context.Background(), state)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if decision.Allow {
		t.Error("expected Allow=false when budget exhausted")
	}
}

func TestPhaseGateRegistry_GetAll(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	gov := NewBudgetGovernor(db)
	registry := NewPhaseGateRegistry(gov)

	phases := []domain.Phase{
		domain.PhaseA, domain.PhaseB, domain.PhaseC,
		domain.PhaseD, domain.PhaseE, domain.PhaseF, domain.PhaseG,
	}
	for _, p := range phases {
		gate, err := registry.Get(p)
		if err != nil {
			t.Errorf("Get(%s): %v", p, err)
		}
		if gate == nil {
			t.Errorf("Get(%s) returned nil gate", p)
		}
		if gate.Name() != "default" {
			t.Errorf("Get(%s).Name() = %q, want %q", p, gate.Name(), "default")
		}
	}
}

func TestPhaseGateRegistry_UnknownPhase(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	gov := NewBudgetGovernor(db)
	registry := NewPhaseGateRegistry(gov)

	_, err = registry.Get(domain.Phase("Z"))
	if err != domain.ErrGateNotRegistered {
		t.Errorf("expected ErrGateNotRegistered, got %v", err)
	}
}
