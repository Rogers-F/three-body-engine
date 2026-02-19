package workflow

import (
	"context"
	"database/sql"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
)

// BudgetGovernor enforces budget limits for workflow tasks.
type BudgetGovernor struct {
	DB       *sql.DB
	TaskRepo *store.TaskRepo

	// WarnRatio is the fraction of budget at which a warning is issued (default 0.8).
	WarnRatio float64
	// HaltRatio is the fraction of budget at which execution is halted (default 1.0).
	HaltRatio float64
}

// NewBudgetGovernor creates a governor with standard thresholds.
func NewBudgetGovernor(db *sql.DB) *BudgetGovernor {
	return &BudgetGovernor{
		DB:        db,
		TaskRepo:  &store.TaskRepo{},
		WarnRatio: 0.8,
		HaltRatio: 1.0,
	}
}

// RecordUsage adds a cost delta to the task's budget and returns the resulting action.
func (g *BudgetGovernor) RecordUsage(ctx context.Context, taskID string, delta domain.CostDelta) (domain.CostAction, error) {
	state, err := g.TaskRepo.GetByID(ctx, g.DB, taskID)
	if err != nil {
		return domain.CostContinue, err
	}

	state.BudgetUsedUSD += delta.AmountUSD

	tx, err := g.DB.BeginTx(ctx, nil)
	if err != nil {
		return domain.CostContinue, err
	}
	defer tx.Rollback()

	if err := g.TaskRepo.UpdateStateTx(ctx, tx, *state); err != nil {
		return domain.CostContinue, err
	}
	if err := tx.Commit(); err != nil {
		return domain.CostContinue, err
	}

	return g.evaluate(state.BudgetUsedUSD, state.BudgetCapUSD), nil
}

// CheckBudget evaluates the current budget status without modifying it.
func (g *BudgetGovernor) CheckBudget(ctx context.Context, state domain.FlowState) (domain.CostAction, error) {
	return g.evaluate(state.BudgetUsedUSD, state.BudgetCapUSD), nil
}

func (g *BudgetGovernor) evaluate(used, cap float64) domain.CostAction {
	if cap <= 0 {
		return domain.CostContinue
	}
	ratio := used / cap
	if ratio >= g.HaltRatio {
		return domain.CostHalt
	}
	if ratio >= g.WarnRatio {
		return domain.CostWarn
	}
	return domain.CostContinue
}
