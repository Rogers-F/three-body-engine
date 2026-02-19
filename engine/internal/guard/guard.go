package guard

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
	"github.com/anthropics/three-body-engine/internal/team"
	"github.com/anthropics/three-body-engine/internal/workflow"
)

// GuardConfig holds rate and round limits.
type GuardConfig struct {
	MaxRounds          int
	RateLimitPerMinute int
}

// Guard coordinates budget, permission, rate, and round checks.
type Guard struct {
	Governor *workflow.BudgetGovernor
	Broker   *team.PermissionBroker
	Config   GuardConfig
	TaskRepo *store.TaskRepo
	DB       *sql.DB

	mu         sync.Mutex
	rateCounts map[string]*rateBucket
}

type rateBucket struct {
	count       int
	windowStart int64
}

// NewGuard creates a Guard with the given dependencies.
func NewGuard(db *sql.DB, gov *workflow.BudgetGovernor, broker *team.PermissionBroker, cfg GuardConfig) *Guard {
	return &Guard{
		Governor:   gov,
		Broker:     broker,
		Config:     cfg,
		TaskRepo:   &store.TaskRepo{},
		DB:         db,
		rateCounts: make(map[string]*rateBucket),
	}
}

// CheckAll runs all checks in order: budget, permission, rate limit, rounds.
// It short-circuits on the first error.
func (g *Guard) CheckAll(ctx context.Context, taskID, path, command string, sheet *domain.CapabilitySheet) error {
	action, err := g.CheckBudget(ctx, taskID)
	if err != nil {
		return err
	}
	if action == domain.CostHalt {
		return domain.ErrBudgetExceeded
	}

	allowed, err := g.Broker.CheckPermission(ctx, sheet, path, command)
	if err != nil {
		return err
	}
	if !allowed {
		return domain.ErrPermissionDenied
	}

	if err := g.CheckRateLimit(taskID); err != nil {
		return err
	}

	if err := g.CheckRounds(ctx, taskID); err != nil {
		return err
	}

	return nil
}

// CheckBudget fetches the task state and delegates to the BudgetGovernor.
// Returns ErrBudgetExceeded if the action is CostHalt.
func (g *Guard) CheckBudget(ctx context.Context, taskID string) (domain.CostAction, error) {
	state, err := g.TaskRepo.GetByID(ctx, g.DB, taskID)
	if err != nil {
		return domain.CostContinue, err
	}
	return g.Governor.CheckBudget(ctx, *state)
}

// CheckRateLimit enforces a per-task sliding window rate limit.
// The window is 60 seconds. If the count exceeds the configured limit,
// ErrRateLimitExceeded is returned.
func (g *Guard) CheckRateLimit(taskID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().Unix()
	bucket, ok := g.rateCounts[taskID]
	if !ok {
		g.rateCounts[taskID] = &rateBucket{count: 1, windowStart: now}
		return nil
	}

	if now-bucket.windowStart > 60 {
		bucket.count = 1
		bucket.windowStart = now
		return nil
	}

	if bucket.count >= g.Config.RateLimitPerMinute {
		return domain.ErrRateLimitExceeded
	}

	bucket.count++
	return nil
}

// CheckRounds reads the task's FlowState and compares the current round
// against the configured maximum. Returns ErrMaxRoundsExceeded if exceeded.
func (g *Guard) CheckRounds(ctx context.Context, taskID string) error {
	state, err := g.TaskRepo.GetByID(ctx, g.DB, taskID)
	if err != nil {
		return err
	}
	if state.Round >= g.Config.MaxRounds {
		return domain.ErrMaxRoundsExceeded
	}
	return nil
}
