// Package workflow implements the Three-Body Engine's 7-phase state machine.
package workflow

import (
	"context"

	"github.com/anthropics/three-body-engine/internal/domain"
)

// Gate evaluates whether a workflow can exit its current phase.
type Gate interface {
	Name() string
	Evaluate(ctx context.Context, state domain.FlowState) (domain.GateDecision, error)
}

// DefaultGate is a basic gate that checks running status and budget.
type DefaultGate struct {
	Governor *BudgetGovernor
}

// Name returns the gate name.
func (g *DefaultGate) Name() string {
	return "default"
}

// Evaluate checks if the flow is running and within budget.
func (g *DefaultGate) Evaluate(ctx context.Context, state domain.FlowState) (domain.GateDecision, error) {
	decision := domain.GateDecision{Allow: true}

	if state.Status != domain.StatusRunning {
		decision.Allow = false
		decision.Blockers = append(decision.Blockers, "flow is not running (status="+string(state.Status)+")")
		return decision, nil
	}

	action, err := g.Governor.CheckBudget(ctx, state)
	if err != nil {
		return decision, err
	}

	if action == domain.CostHalt {
		decision.Allow = false
		decision.Blockers = append(decision.Blockers, "budget limit exceeded")
		return decision, nil
	}

	return decision, nil
}

// PhaseGateRegistry maps each phase to its gate implementation.
type PhaseGateRegistry struct {
	gates map[domain.Phase]Gate
}

// NewPhaseGateRegistry creates a registry with a default gate for all phases.
func NewPhaseGateRegistry(gov *BudgetGovernor) *PhaseGateRegistry {
	defaultGate := &DefaultGate{Governor: gov}
	gates := map[domain.Phase]Gate{
		domain.PhaseA: defaultGate,
		domain.PhaseB: defaultGate,
		domain.PhaseC: defaultGate,
		domain.PhaseD: defaultGate,
		domain.PhaseE: defaultGate,
		domain.PhaseF: defaultGate,
		domain.PhaseG: defaultGate,
	}
	return &PhaseGateRegistry{gates: gates}
}

// Register sets a custom gate for a phase.
func (r *PhaseGateRegistry) Register(phase domain.Phase, gate Gate) {
	r.gates[phase] = gate
}

// Get returns the gate for a phase, or an error if none is registered.
func (r *PhaseGateRegistry) Get(phase domain.Phase) (Gate, error) {
	g, ok := r.gates[phase]
	if !ok {
		return nil, domain.ErrGateNotRegistered
	}
	return g, nil
}
