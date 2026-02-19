// Package workflow implements the Three-Body Engine's 7-phase state machine.
package workflow

import (
	"context"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/team"
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

// CompactionGate wraps an inner gate and validates compaction slots.
type CompactionGate struct {
	Inner     Gate
	Validator *team.CompactionValidator
	SlotsFn   func(ctx context.Context, state domain.FlowState) (domain.CompactionSlots, error)
}

// Name returns the gate name.
func (g *CompactionGate) Name() string {
	return "compaction"
}

// Evaluate checks the inner gate first, then validates compaction slots.
func (g *CompactionGate) Evaluate(ctx context.Context, state domain.FlowState) (domain.GateDecision, error) {
	inner, err := g.Inner.Evaluate(ctx, state)
	if err != nil {
		return inner, err
	}
	if !inner.Allow {
		return inner, nil
	}

	slots, err := g.SlotsFn(ctx, state)
	if err != nil {
		return domain.GateDecision{}, err
	}

	if vErr := g.Validator.Validate(ctx, slots); vErr != nil {
		return domain.GateDecision{
			Allow:    false,
			Blockers: []string{vErr.Error()},
		}, nil
	}

	return inner, nil
}

// ReviewGate wraps an inner gate and checks for unresolved blockers.
type ReviewGate struct {
	Inner      Gate
	BlockersFn func(ctx context.Context, state domain.FlowState) ([]string, error)
}

// Name returns the gate name.
func (g *ReviewGate) Name() string {
	return "review"
}

// Evaluate checks the inner gate first, then checks for unresolved blockers.
func (g *ReviewGate) Evaluate(ctx context.Context, state domain.FlowState) (domain.GateDecision, error) {
	inner, err := g.Inner.Evaluate(ctx, state)
	if err != nil {
		return inner, err
	}
	if !inner.Allow {
		return inner, nil
	}

	blockers, err := g.BlockersFn(ctx, state)
	if err != nil {
		return domain.GateDecision{}, err
	}

	if len(blockers) > 0 {
		return domain.GateDecision{
			Allow:    false,
			Blockers: blockers,
		}, nil
	}

	return inner, nil
}

// CompositeGate chains multiple gates, evaluating all and aggregating blockers.
type CompositeGate struct {
	Gates []Gate
}

// Name returns the gate name.
func (g *CompositeGate) Name() string {
	return "composite"
}

// Evaluate runs all gates and collects all blockers. Allow is true only if all gates allow.
func (g *CompositeGate) Evaluate(ctx context.Context, state domain.FlowState) (domain.GateDecision, error) {
	result := domain.GateDecision{Allow: true}

	for _, gate := range g.Gates {
		decision, err := gate.Evaluate(ctx, state)
		if err != nil {
			return domain.GateDecision{}, err
		}
		if !decision.Allow {
			result.Allow = false
			result.Blockers = append(result.Blockers, decision.Blockers...)
		}
	}

	return result, nil
}
