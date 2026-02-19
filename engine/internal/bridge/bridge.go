// Package bridge connects the workflow engine to code agent sessions,
// coordinating budget checks, session lifecycle, and cost event processing.
package bridge

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/guard"
	"github.com/anthropics/three-body-engine/internal/mcp"
	"github.com/anthropics/three-body-engine/internal/store"
	"github.com/anthropics/three-body-engine/internal/workflow"
)

// Bridge is the integration layer between the engine and code agent sessions.
type Bridge struct {
	Sessions      *mcp.SessionManager
	Guard         *guard.Guard
	Governor      *workflow.BudgetGovernor
	CostDeltaRepo *store.CostDeltaRepo
	AuditRepo     *store.AuditRepo
	DB            *sql.DB
}

// NewBridge creates a Bridge with all required dependencies.
func NewBridge(
	sessions *mcp.SessionManager,
	g *guard.Guard,
	gov *workflow.BudgetGovernor,
	costDeltaRepo *store.CostDeltaRepo,
	auditRepo *store.AuditRepo,
	db *sql.DB,
) *Bridge {
	return &Bridge{
		Sessions:      sessions,
		Guard:         g,
		Governor:      gov,
		CostDeltaRepo: costDeltaRepo,
		AuditRepo:     auditRepo,
		DB:            db,
	}
}

// StartSession checks the budget guard, creates a code agent session, and logs an audit record.
func (b *Bridge) StartSession(ctx context.Context, worker domain.WorkerRef, cfg domain.SessionConfig) (string, error) {
	action, err := b.Guard.CheckBudget(ctx, worker.TaskID)
	if err != nil {
		return "", fmt.Errorf("bridge start session: budget check: %w", err)
	}
	if action == domain.CostHalt {
		return "", domain.ErrBudgetExceeded
	}

	sessionID, err := b.Sessions.Create(ctx, domain.Provider(worker.Role), cfg)
	if err != nil {
		return "", fmt.Errorf("bridge start session: create: %w", err)
	}

	_ = b.AuditRepo.Record(ctx, b.DB, domain.AuditRecord{
		ID:        fmt.Sprintf("aud-start-%s-%d", sessionID, time.Now().UnixNano()),
		TaskID:    worker.TaskID,
		Category:  "session",
		Actor:     "bridge",
		Action:    "start_session",
		RequestJSON: mustJSON(map[string]string{
			"session_id": sessionID,
			"worker_id":  worker.WorkerID,
			"role":       worker.Role,
		}),
		DecisionJSON: mustJSON(map[string]string{"result": "started"}),
		Severity:     "info",
		CreatedAt:    time.Now().Unix(),
	})

	return sessionID, nil
}

// StopSession terminates a session and logs an audit record.
// Process kill errors (e.g., already exited) are ignored since the session
// is still removed from the manager regardless.
func (b *Bridge) StopSession(ctx context.Context, sessionID string) error {
	sess, err := b.Sessions.Get(sessionID)
	if err != nil {
		return err
	}

	taskID := sess.Config.TaskID

	// Stop removes the session from the manager and kills the process.
	// On Windows, killing an already-exited process returns an error; we
	// treat that as a successful stop since the session is cleaned up.
	_ = b.Sessions.Stop(sessionID)

	_ = b.AuditRepo.Record(ctx, b.DB, domain.AuditRecord{
		ID:        fmt.Sprintf("aud-stop-%s-%d", sessionID, time.Now().UnixNano()),
		TaskID:    taskID,
		Category:  "session",
		Actor:     "bridge",
		Action:    "stop_session",
		RequestJSON: mustJSON(map[string]string{
			"session_id": sessionID,
		}),
		DecisionJSON: mustJSON(map[string]string{"result": "stopped"}),
		Severity:     "info",
		CreatedAt:    time.Now().Unix(),
	})

	return nil
}

// StreamEvents returns a channel that forwards events from a session.
// Cost events (Type=="cost") are automatically recorded via the BudgetGovernor and CostDeltaRepo.
func (b *Bridge) StreamEvents(ctx context.Context, sessionID string) (<-chan domain.NormalizedEvent, error) {
	sess, err := b.Sessions.Get(sessionID)
	if err != nil {
		return nil, err
	}

	out := make(chan domain.NormalizedEvent, 64)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-sess.Events():
				if !ok {
					return
				}
				if ev.Type == "cost" {
					b.processCostEvent(ctx, sess.Config.TaskID, ev)
				}
				select {
				case out <- ev:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}

// processCostEvent extracts a CostDelta from the event payload and records it.
func (b *Bridge) processCostEvent(ctx context.Context, taskID string, ev domain.NormalizedEvent) {
	var delta domain.CostDelta
	if err := json.Unmarshal(ev.Payload, &delta); err != nil {
		return
	}
	delta.Provider = ev.Provider
	delta.CreatedAt = time.Now().Unix()

	_, _ = b.Governor.RecordUsage(ctx, taskID, delta)
	_ = b.CostDeltaRepo.Create(ctx, b.DB, taskID, delta)
}

// mustJSON marshals v to a JSON string, returning "{}" on error.
func mustJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
