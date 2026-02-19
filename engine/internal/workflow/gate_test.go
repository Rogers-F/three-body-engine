package workflow

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
	"github.com/anthropics/three-body-engine/internal/team"
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

// --- Helper: always-allow gate for testing composite gates ---

type stubGate struct {
	name     string
	allow    bool
	blockers []string
	err      error
}

func (g *stubGate) Name() string { return g.name }
func (g *stubGate) Evaluate(_ context.Context, _ domain.FlowState) (domain.GateDecision, error) {
	if g.err != nil {
		return domain.GateDecision{}, g.err
	}
	return domain.GateDecision{Allow: g.allow, Blockers: g.blockers}, nil
}

// --- CompactionGate tests ---

func TestCompactionGate_PassesWhenBothPass(t *testing.T) {
	inner := &stubGate{name: "inner", allow: true}
	validator := &team.CompactionValidator{}
	gate := &CompactionGate{
		Inner:     inner,
		Validator: validator,
		SlotsFn: func(_ context.Context, _ domain.FlowState) (domain.CompactionSlots, error) {
			return domain.CompactionSlots{
				TaskSpec:           "spec",
				AcceptanceCriteria: "criteria",
				CurrentPhase:       "C",
				ArtifactRefs:       []domain.ArtifactRef{{ID: "a1"}},
			}, nil
		},
	}

	decision, err := gate.Evaluate(context.Background(), domain.FlowState{Status: domain.StatusRunning})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !decision.Allow {
		t.Errorf("expected Allow=true, got false; blockers: %v", decision.Blockers)
	}
}

func TestCompactionGate_BlocksWhenCompactionFails(t *testing.T) {
	inner := &stubGate{name: "inner", allow: true}
	validator := &team.CompactionValidator{}
	gate := &CompactionGate{
		Inner:     inner,
		Validator: validator,
		SlotsFn: func(_ context.Context, _ domain.FlowState) (domain.CompactionSlots, error) {
			return domain.CompactionSlots{
				TaskSpec: "spec",
				// Missing AcceptanceCriteria, CurrentPhase, ArtifactRefs
			}, nil
		},
	}

	decision, err := gate.Evaluate(context.Background(), domain.FlowState{Status: domain.StatusRunning})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if decision.Allow {
		t.Error("expected Allow=false when compaction fails")
	}
	if len(decision.Blockers) == 0 {
		t.Error("expected blockers from compaction failure")
	}
}

func TestCompactionGate_BlocksWhenInnerBlocks(t *testing.T) {
	inner := &stubGate{name: "inner", allow: false, blockers: []string{"inner blocked"}}
	validator := &team.CompactionValidator{}
	slotsCalled := false
	gate := &CompactionGate{
		Inner:     inner,
		Validator: validator,
		SlotsFn: func(_ context.Context, _ domain.FlowState) (domain.CompactionSlots, error) {
			slotsCalled = true
			return domain.CompactionSlots{}, nil
		},
	}

	decision, err := gate.Evaluate(context.Background(), domain.FlowState{Status: domain.StatusBlocked})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if decision.Allow {
		t.Error("expected Allow=false when inner blocks")
	}
	if slotsCalled {
		t.Error("SlotsFn should not be called when inner gate blocks")
	}
}

// --- ReviewGate tests ---

func TestReviewGate_PassesWithNoBlockers(t *testing.T) {
	inner := &stubGate{name: "inner", allow: true}
	gate := &ReviewGate{
		Inner: inner,
		BlockersFn: func(_ context.Context, _ domain.FlowState) ([]string, error) {
			return nil, nil
		},
	}

	decision, err := gate.Evaluate(context.Background(), domain.FlowState{Status: domain.StatusRunning})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !decision.Allow {
		t.Errorf("expected Allow=true, got false; blockers: %v", decision.Blockers)
	}
}

func TestReviewGate_BlocksWithP0Blockers(t *testing.T) {
	inner := &stubGate{name: "inner", allow: true}
	gate := &ReviewGate{
		Inner: inner,
		BlockersFn: func(_ context.Context, _ domain.FlowState) ([]string, error) {
			return []string{"P0: critical security issue", "P1: missing error handling"}, nil
		},
	}

	decision, err := gate.Evaluate(context.Background(), domain.FlowState{Status: domain.StatusRunning})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if decision.Allow {
		t.Error("expected Allow=false with P0 blockers")
	}
	if len(decision.Blockers) != 2 {
		t.Errorf("expected 2 blockers, got %d", len(decision.Blockers))
	}
}

// --- CompositeGate tests ---

func TestCompositeGate_AllPass(t *testing.T) {
	gate := &CompositeGate{
		Gates: []Gate{
			&stubGate{name: "a", allow: true},
			&stubGate{name: "b", allow: true},
			&stubGate{name: "c", allow: true},
		},
	}

	decision, err := gate.Evaluate(context.Background(), domain.FlowState{Status: domain.StatusRunning})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !decision.Allow {
		t.Errorf("expected Allow=true, got false; blockers: %v", decision.Blockers)
	}
}

func TestCompositeGate_AggregatesBlockers(t *testing.T) {
	gate := &CompositeGate{
		Gates: []Gate{
			&stubGate{name: "a", allow: false, blockers: []string{"blocker from a"}},
			&stubGate{name: "b", allow: true},
			&stubGate{name: "c", allow: false, blockers: []string{"blocker from c1", "blocker from c2"}},
		},
	}

	decision, err := gate.Evaluate(context.Background(), domain.FlowState{Status: domain.StatusRunning})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if decision.Allow {
		t.Error("expected Allow=false when any gate blocks")
	}
	if len(decision.Blockers) != 3 {
		t.Errorf("expected 3 aggregated blockers, got %d: %v", len(decision.Blockers), decision.Blockers)
	}
}

func TestCompositeGate_PropagatesError(t *testing.T) {
	testErr := errors.New("test error")
	gate := &CompositeGate{
		Gates: []Gate{
			&stubGate{name: "a", allow: true},
			&stubGate{name: "b", err: testErr},
		},
	}

	_, err := gate.Evaluate(context.Background(), domain.FlowState{Status: domain.StatusRunning})
	if err == nil {
		t.Fatal("expected error from gate evaluation")
	}
	if !errors.Is(err, testErr) {
		t.Errorf("expected testErr, got %v", err)
	}
}
