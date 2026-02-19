package workflow

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
)

// validTransitions defines the legal phase transitions.
// Each key is a source phase, and the value is the set of valid target phases.
var validTransitions = map[domain.Phase]map[domain.Phase]bool{
	domain.PhaseA: {domain.PhaseB: true},
	domain.PhaseB: {domain.PhaseC: true},
	domain.PhaseC: {domain.PhaseD: true},
	domain.PhaseD: {domain.PhaseE: true, domain.PhaseC: true}, // D->C is rollback
	domain.PhaseE: {domain.PhaseF: true},
	domain.PhaseF: {domain.PhaseG: true, domain.PhaseE: true}, // F->E is rework
}

// IsValidTransition checks if a phase transition is legal.
func IsValidTransition(from, to domain.Phase) bool {
	targets, ok := validTransitions[from]
	if !ok {
		return false
	}
	return targets[to]
}

// Engine is the FSM that manages workflow state transitions.
type Engine struct {
	DB           *sql.DB
	TaskRepo     *store.TaskRepo
	EventRepo    *store.EventRepo
	SnapshotRepo *store.SnapshotRepo
	GateRegistry *PhaseGateRegistry
}

// NewEngine creates a new FSM engine with all dependencies.
func NewEngine(db *sql.DB) *Engine {
	gov := NewBudgetGovernor(db)
	return &Engine{
		DB:           db,
		TaskRepo:     &store.TaskRepo{},
		EventRepo:    &store.EventRepo{},
		SnapshotRepo: &store.SnapshotRepo{},
		GateRegistry: NewPhaseGateRegistry(gov),
	}
}

// StartFlow creates a new workflow at Phase A with the given budget cap.
func (e *Engine) StartFlow(ctx context.Context, taskID string, budgetCapUSD float64) error {
	state := domain.FlowState{
		TaskID:        taskID,
		CurrentPhase:  domain.PhaseA,
		Status:        domain.StatusRunning,
		StateVersion:  1,
		Round:         0,
		BudgetCapUSD:  budgetCapUSD,
		BudgetUsedUSD: 0,
		LastEventSeq:  1, // The initial flow_started event uses seq 1.
		UpdatedAtUnix: time.Now().Unix(),
	}

	tx, err := e.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := e.TaskRepo.CreateTx(ctx, tx, state); err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	now := time.Now().Unix()
	event := domain.WorkflowEvent{
		TaskID:      taskID,
		SeqNo:       1,
		Phase:       domain.PhaseA,
		EventType:   "flow_started",
		PayloadJSON: "{}",
		CreatedAt:   now,
	}
	if err := e.EventRepo.AppendTx(ctx, tx, event); err != nil {
		return fmt.Errorf("append start event: %w", err)
	}

	return tx.Commit()
}

// Advance moves a workflow to the next phase based on the trigger.
// The entire transition is performed in a single transaction with optimistic locking.
func (e *Engine) Advance(ctx context.Context, taskID string, trigger domain.TransitionTrigger) error {
	// Load current state.
	state, err := e.TaskRepo.GetByID(ctx, e.DB, taskID)
	if err != nil {
		return err
	}

	if state.Status == domain.StatusDone {
		return domain.ErrFlowAlreadyDone
	}

	// Evaluate the gate for the current phase.
	gate, err := e.GateRegistry.Get(state.CurrentPhase)
	if err != nil {
		return err
	}

	decision, err := gate.Evaluate(ctx, *state)
	if err != nil {
		return fmt.Errorf("evaluate gate: %w", err)
	}

	if !decision.Allow {
		return domain.NewEngineError(
			domain.ErrPhaseGateFailed.Code,
			fmt.Sprintf("gate blocked transition: %v", decision.Blockers),
		)
	}

	// Determine the target phase from the trigger action.
	nextPhase, err := resolveNextPhase(state.CurrentPhase, trigger.Action)
	if err != nil {
		return err
	}

	// Validate the transition is legal.
	if !IsValidTransition(state.CurrentPhase, nextPhase) {
		return domain.NewEngineError(
			domain.ErrInvalidTransition.Code,
			fmt.Sprintf("illegal transition %s -> %s", state.CurrentPhase, nextPhase),
		)
	}

	// Perform the transition in a single transaction.
	tx, err := e.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().Unix()
	newSeq := state.LastEventSeq + 1

	// Append the transition event.
	event := domain.WorkflowEvent{
		TaskID:      taskID,
		SeqNo:       newSeq,
		Phase:       nextPhase,
		EventType:   "phase_transition",
		PayloadJSON: fmt.Sprintf(`{"from":"%s","to":"%s","action":"%s","actor":"%s"}`, state.CurrentPhase, nextPhase, trigger.Action, trigger.Actor),
		CreatedAt:   now,
	}
	if err := e.EventRepo.AppendTx(ctx, tx, event); err != nil {
		return fmt.Errorf("append transition event: %w", err)
	}

	// Save a snapshot at the phase boundary.
	snap := domain.PhaseSnapshot{
		TaskID:       taskID,
		Phase:        nextPhase,
		Round:        state.Round,
		SnapshotJSON: fmt.Sprintf(`{"from_phase":"%s","to_phase":"%s","trigger":"%s"}`, state.CurrentPhase, nextPhase, trigger.Action),
		Checksum:     "",
		CreatedAt:    now,
	}
	if err := e.SnapshotRepo.SaveTx(ctx, tx, snap); err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}

	// Update the state with optimistic locking.
	updatedState := *state
	updatedState.CurrentPhase = nextPhase
	updatedState.LastEventSeq = newSeq
	updatedState.UpdatedAtUnix = now

	// If transitioning to phase G, mark as done.
	if nextPhase == domain.PhaseG {
		updatedState.Status = domain.StatusDone
	}

	// Track rollback/rework rounds.
	if (state.CurrentPhase == domain.PhaseD && nextPhase == domain.PhaseC) ||
		(state.CurrentPhase == domain.PhaseF && nextPhase == domain.PhaseE) {
		updatedState.Round = state.Round + 1
	}

	if err := e.TaskRepo.UpdateStateTx(ctx, tx, updatedState); err != nil {
		return err
	}

	return tx.Commit()
}

// GetState returns the current state of a workflow.
func (e *Engine) GetState(ctx context.Context, taskID string) (*domain.FlowState, error) {
	return e.TaskRepo.GetByID(ctx, e.DB, taskID)
}

// resolveNextPhase determines the target phase from the trigger action.
func resolveNextPhase(current domain.Phase, action string) (domain.Phase, error) {
	switch action {
	case "advance":
		return nextPhaseForward(current)
	case "rollback":
		if current == domain.PhaseD {
			return domain.PhaseC, nil
		}
		return "", domain.NewEngineError(
			domain.ErrInvalidTransition.Code,
			fmt.Sprintf("rollback not allowed from phase %s", current),
		)
	case "rework":
		if current == domain.PhaseF {
			return domain.PhaseE, nil
		}
		return "", domain.NewEngineError(
			domain.ErrInvalidTransition.Code,
			fmt.Sprintf("rework not allowed from phase %s", current),
		)
	default:
		return "", domain.NewEngineError(
			domain.ErrInvalidTransition.Code,
			fmt.Sprintf("unknown action: %s", action),
		)
	}
}

// nextPhaseForward returns the next phase in the standard forward path.
func nextPhaseForward(current domain.Phase) (domain.Phase, error) {
	switch current {
	case domain.PhaseA:
		return domain.PhaseB, nil
	case domain.PhaseB:
		return domain.PhaseC, nil
	case domain.PhaseC:
		return domain.PhaseD, nil
	case domain.PhaseD:
		return domain.PhaseE, nil
	case domain.PhaseE:
		return domain.PhaseF, nil
	case domain.PhaseF:
		return domain.PhaseG, nil
	default:
		return "", domain.NewEngineError(
			domain.ErrInvalidTransition.Code,
			fmt.Sprintf("no forward transition from phase %s", current),
		)
	}
}
