package workflow

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
)

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewEngine(db)
}

func TestEngine_StartFlow(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	if err := eng.StartFlow(ctx, "task-1", 10.0); err != nil {
		t.Fatalf("StartFlow: %v", err)
	}

	state, err := eng.GetState(ctx, "task-1")
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if state.CurrentPhase != domain.PhaseA {
		t.Errorf("Phase = %q, want A", state.CurrentPhase)
	}
	if state.Status != domain.StatusRunning {
		t.Errorf("Status = %q, want running", state.Status)
	}
	if state.BudgetCapUSD != 10.0 {
		t.Errorf("BudgetCapUSD = %f, want 10.0", state.BudgetCapUSD)
	}
}

func TestEngine_StartFlow_Duplicate(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	if err := eng.StartFlow(ctx, "task-1", 10.0); err != nil {
		t.Fatalf("first StartFlow: %v", err)
	}
	if err := eng.StartFlow(ctx, "task-1", 10.0); err == nil {
		t.Error("expected error on duplicate StartFlow, got nil")
	}
}

func TestEngine_AdvanceForward_FullPath(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	if err := eng.StartFlow(ctx, "task-1", 100.0); err != nil {
		t.Fatalf("StartFlow: %v", err)
	}

	trigger := domain.TransitionTrigger{Action: "advance", Actor: "test"}

	// Walk through the full path: A -> B -> C -> D -> E -> F -> G
	expectedPhases := []domain.Phase{
		domain.PhaseB, domain.PhaseC, domain.PhaseD,
		domain.PhaseE, domain.PhaseF, domain.PhaseG,
	}

	for _, expected := range expectedPhases {
		if err := eng.Advance(ctx, "task-1", trigger); err != nil {
			t.Fatalf("Advance to %s: %v", expected, err)
		}
		state, err := eng.GetState(ctx, "task-1")
		if err != nil {
			t.Fatalf("GetState: %v", err)
		}
		if state.CurrentPhase != expected {
			t.Errorf("Phase = %q, want %q", state.CurrentPhase, expected)
		}
	}

	// Verify final state is done.
	state, err := eng.GetState(ctx, "task-1")
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if state.Status != domain.StatusDone {
		t.Errorf("Status = %q, want completed", state.Status)
	}
}

func TestEngine_AdvanceFromDone(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	eng.StartFlow(ctx, "task-1", 100.0)

	trigger := domain.TransitionTrigger{Action: "advance", Actor: "test"}
	// Advance to G (done).
	for i := 0; i < 6; i++ {
		eng.Advance(ctx, "task-1", trigger)
	}

	// Should fail because flow is already done.
	err := eng.Advance(ctx, "task-1", trigger)
	if err == nil {
		t.Error("expected error advancing a done flow, got nil")
	}
}

func TestEngine_Rollback_D_to_C(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	eng.StartFlow(ctx, "task-1", 100.0)
	advanceTrigger := domain.TransitionTrigger{Action: "advance", Actor: "test"}

	// Advance to D: A -> B -> C -> D
	for i := 0; i < 3; i++ {
		if err := eng.Advance(ctx, "task-1", advanceTrigger); err != nil {
			t.Fatalf("Advance step %d: %v", i, err)
		}
	}

	state, _ := eng.GetState(ctx, "task-1")
	if state.CurrentPhase != domain.PhaseD {
		t.Fatalf("expected phase D, got %s", state.CurrentPhase)
	}

	// Rollback D -> C.
	rollback := domain.TransitionTrigger{Action: "rollback", Actor: "test"}
	if err := eng.Advance(ctx, "task-1", rollback); err != nil {
		t.Fatalf("Rollback D->C: %v", err)
	}

	state, _ = eng.GetState(ctx, "task-1")
	if state.CurrentPhase != domain.PhaseC {
		t.Errorf("Phase = %q after rollback, want C", state.CurrentPhase)
	}
	if state.Round != 1 {
		t.Errorf("Round = %d after rollback, want 1", state.Round)
	}
}

func TestEngine_Rework_F_to_E(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	eng.StartFlow(ctx, "task-1", 100.0)
	advanceTrigger := domain.TransitionTrigger{Action: "advance", Actor: "test"}

	// Advance to F: A -> B -> C -> D -> E -> F
	for i := 0; i < 5; i++ {
		if err := eng.Advance(ctx, "task-1", advanceTrigger); err != nil {
			t.Fatalf("Advance step %d: %v", i, err)
		}
	}

	state, _ := eng.GetState(ctx, "task-1")
	if state.CurrentPhase != domain.PhaseF {
		t.Fatalf("expected phase F, got %s", state.CurrentPhase)
	}

	// Rework F -> E.
	rework := domain.TransitionTrigger{Action: "rework", Actor: "test"}
	if err := eng.Advance(ctx, "task-1", rework); err != nil {
		t.Fatalf("Rework F->E: %v", err)
	}

	state, _ = eng.GetState(ctx, "task-1")
	if state.CurrentPhase != domain.PhaseE {
		t.Errorf("Phase = %q after rework, want E", state.CurrentPhase)
	}
	if state.Round != 1 {
		t.Errorf("Round = %d after rework, want 1", state.Round)
	}
}

func TestEngine_InvalidTransition_RollbackFromB(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	eng.StartFlow(ctx, "task-1", 100.0)
	eng.Advance(ctx, "task-1", domain.TransitionTrigger{Action: "advance", Actor: "test"})

	state, _ := eng.GetState(ctx, "task-1")
	if state.CurrentPhase != domain.PhaseB {
		t.Fatalf("expected phase B, got %s", state.CurrentPhase)
	}

	// Rollback from B should not be allowed.
	err := eng.Advance(ctx, "task-1", domain.TransitionTrigger{Action: "rollback", Actor: "test"})
	if err == nil {
		t.Error("expected error on rollback from B, got nil")
	}
}

func TestEngine_InvalidTransition_ReworkFromD(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	eng.StartFlow(ctx, "task-1", 100.0)
	advanceTrigger := domain.TransitionTrigger{Action: "advance", Actor: "test"}
	for i := 0; i < 3; i++ {
		eng.Advance(ctx, "task-1", advanceTrigger)
	}

	state, _ := eng.GetState(ctx, "task-1")
	if state.CurrentPhase != domain.PhaseD {
		t.Fatalf("expected phase D, got %s", state.CurrentPhase)
	}

	// Rework from D should not be allowed (only from F).
	err := eng.Advance(ctx, "task-1", domain.TransitionTrigger{Action: "rework", Actor: "test"})
	if err == nil {
		t.Error("expected error on rework from D, got nil")
	}
}

func TestEngine_GetState_NotFound(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	_, err := eng.GetState(ctx, "nonexistent")
	if err != domain.ErrFlowNotFound {
		t.Errorf("expected ErrFlowNotFound, got %v", err)
	}
}

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		from  domain.Phase
		to    domain.Phase
		valid bool
	}{
		{domain.PhaseA, domain.PhaseB, true},
		{domain.PhaseB, domain.PhaseC, true},
		{domain.PhaseC, domain.PhaseD, true},
		{domain.PhaseD, domain.PhaseE, true},
		{domain.PhaseD, domain.PhaseC, true},  // rollback
		{domain.PhaseE, domain.PhaseF, true},
		{domain.PhaseF, domain.PhaseG, true},
		{domain.PhaseF, domain.PhaseE, true},  // rework
		// Invalid transitions:
		{domain.PhaseA, domain.PhaseC, false},
		{domain.PhaseB, domain.PhaseA, false},
		{domain.PhaseC, domain.PhaseA, false},
		{domain.PhaseE, domain.PhaseC, false},
		{domain.PhaseG, domain.PhaseA, false},
		{domain.PhaseA, domain.PhaseG, false},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("%s->%s", tt.from, tt.to)
		t.Run(name, func(t *testing.T) {
			got := IsValidTransition(tt.from, tt.to)
			if got != tt.valid {
				t.Errorf("IsValidTransition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.valid)
			}
		})
	}
}
