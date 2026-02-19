package bridge

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/guard"
	"github.com/anthropics/three-body-engine/internal/mcp"
	"github.com/anthropics/three-body-engine/internal/store"
	"github.com/anthropics/three-body-engine/internal/team"
	"github.com/anthropics/three-body-engine/internal/workflow"
)

// testHarness sets up a Bridge backed by a real SQLite database.
type testHarness struct {
	Bridge *Bridge
	DB     *store.TaskRepo
}

func newHarness(t *testing.T) *testHarness {
	t.Helper()

	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	reg := mcp.NewProviderRegistry()
	cmd, args := echoCommand()
	if err := reg.Register(mcp.ProviderSpec{
		Name:    domain.ProviderClaude,
		Command: cmd,
		Args:    args,
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	sessions := mcp.NewSessionManager(reg)
	t.Cleanup(func() { sessions.StopAll() })

	gov := workflow.NewBudgetGovernor(db)
	broker := team.NewPermissionBroker(db)
	g := guard.NewGuard(db, gov, broker, guard.GuardConfig{
		MaxRounds:          10,
		RateLimitPerMinute: 100,
	})

	b := NewBridge(sessions, g, gov, &store.CostDeltaRepo{}, &store.AuditRepo{}, db)

	return &testHarness{Bridge: b, DB: &store.TaskRepo{}}
}

// createTask inserts a task with a budget into the test DB.
func (h *testHarness) createTask(t *testing.T, taskID string, budgetCap float64) {
	t.Helper()
	tx, err := h.Bridge.DB.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	state := domain.FlowState{
		TaskID:       taskID,
		CurrentPhase: domain.PhaseA,
		Status:       domain.StatusRunning,
		StateVersion: 1,
		BudgetCapUSD: budgetCap,
	}
	if err := h.DB.CreateTx(context.Background(), tx, state); err != nil {
		tx.Rollback()
		t.Fatalf("create task: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func echoCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", `echo {"type":"result","data":"ok"}`}
	}
	return "sh", []string{"-c", `echo '{"type":"result","data":"ok"}'`}
}

// ---------------------------------------------------------------------------
// StartSession tests
// ---------------------------------------------------------------------------

func TestStartSession_Success(t *testing.T) {
	h := newHarness(t)
	h.createTask(t, "task-start", 100.0)

	ctx := context.Background()
	worker := domain.WorkerRef{
		WorkerID: "w-1",
		TaskID:   "task-start",
		Role:     string(domain.ProviderClaude),
	}
	cfg := domain.SessionConfig{
		TaskID:    "task-start",
		Role:      string(domain.ProviderClaude),
		Workspace: t.TempDir(),
	}

	sessionID, err := h.Bridge.StartSession(ctx, worker, cfg)
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	if sessionID == "" {
		t.Fatal("expected non-empty session ID")
	}
}

func TestStartSession_GuardDenied(t *testing.T) {
	h := newHarness(t)
	// Create a task that is already over budget.
	h.createTask(t, "task-over", 0.01)

	// Record usage to exceed the budget.
	_, err := h.Bridge.Governor.RecordUsage(context.Background(), "task-over", domain.CostDelta{AmountUSD: 10.0})
	if err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}

	ctx := context.Background()
	worker := domain.WorkerRef{
		WorkerID: "w-2",
		TaskID:   "task-over",
		Role:     string(domain.ProviderClaude),
	}
	cfg := domain.SessionConfig{TaskID: "task-over", Role: string(domain.ProviderClaude), Workspace: t.TempDir()}

	_, err = h.Bridge.StartSession(ctx, worker, cfg)
	if err == nil {
		t.Fatal("expected error for exceeded budget, got nil")
	}
}

func TestStartSession_AuditsAction(t *testing.T) {
	h := newHarness(t)
	h.createTask(t, "task-audit-start", 100.0)

	ctx := context.Background()
	worker := domain.WorkerRef{
		WorkerID: "w-aud",
		TaskID:   "task-audit-start",
		Role:     string(domain.ProviderClaude),
	}
	cfg := domain.SessionConfig{TaskID: "task-audit-start", Role: string(domain.ProviderClaude), Workspace: t.TempDir()}

	_, err := h.Bridge.StartSession(ctx, worker, cfg)
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}

	// Verify an audit record was created.
	records, err := h.Bridge.AuditRepo.ListByTask(ctx, h.Bridge.DB, "task-audit-start")
	if err != nil {
		t.Fatalf("ListByTask: %v", err)
	}
	if len(records) == 0 {
		t.Error("expected at least one audit record after StartSession")
	}
	found := false
	for _, r := range records {
		if r.Action == "start_session" {
			found = true
			break
		}
	}
	if !found {
		t.Error("no audit record with action=start_session found")
	}
}

// ---------------------------------------------------------------------------
// StopSession tests
// ---------------------------------------------------------------------------

func TestStopSession_Success(t *testing.T) {
	h := newHarness(t)
	h.createTask(t, "task-stop", 100.0)

	ctx := context.Background()
	worker := domain.WorkerRef{
		WorkerID: "w-stop",
		TaskID:   "task-stop",
		Role:     string(domain.ProviderClaude),
	}
	cfg := domain.SessionConfig{TaskID: "task-stop", Role: string(domain.ProviderClaude), Workspace: t.TempDir()}

	sessionID, err := h.Bridge.StartSession(ctx, worker, cfg)
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}

	// Allow the echo process to finish.
	time.Sleep(200 * time.Millisecond)

	err = h.Bridge.StopSession(ctx, sessionID)
	if err != nil {
		// On Windows, killing an already-exited process can return an error.
		t.Logf("StopSession returned (may be expected): %v", err)
	}
}

func TestStopSession_NotFound(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	err := h.Bridge.StopSession(ctx, "nonexistent-session")
	if err == nil {
		t.Fatal("expected error for nonexistent session, got nil")
	}
}

func TestStopSession_AuditsAction(t *testing.T) {
	h := newHarness(t)
	h.createTask(t, "task-stop-aud", 100.0)

	ctx := context.Background()
	worker := domain.WorkerRef{
		WorkerID: "w-stop-aud",
		TaskID:   "task-stop-aud",
		Role:     string(domain.ProviderClaude),
	}
	cfg := domain.SessionConfig{TaskID: "task-stop-aud", Role: string(domain.ProviderClaude), Workspace: t.TempDir()}

	sessionID, err := h.Bridge.StartSession(ctx, worker, cfg)
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	_ = h.Bridge.StopSession(ctx, sessionID)

	records, err := h.Bridge.AuditRepo.ListByTask(ctx, h.Bridge.DB, "task-stop-aud")
	if err != nil {
		t.Fatalf("ListByTask: %v", err)
	}

	found := false
	for _, r := range records {
		if r.Action == "stop_session" {
			found = true
			break
		}
	}
	if !found {
		t.Error("no audit record with action=stop_session found")
	}
}

// ---------------------------------------------------------------------------
// StreamEvents tests
// ---------------------------------------------------------------------------

func TestStreamEvents_ReturnsChannel(t *testing.T) {
	h := newHarness(t)
	h.createTask(t, "task-stream", 100.0)

	ctx := context.Background()
	worker := domain.WorkerRef{
		WorkerID: "w-stream",
		TaskID:   "task-stream",
		Role:     string(domain.ProviderClaude),
	}
	cfg := domain.SessionConfig{TaskID: "task-stream", Role: string(domain.ProviderClaude), Workspace: t.TempDir()}

	sessionID, err := h.Bridge.StartSession(ctx, worker, cfg)
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}

	ch, err := h.Bridge.StreamEvents(ctx, sessionID)
	if err != nil {
		t.Fatalf("StreamEvents: %v", err)
	}
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	// The echo command outputs one JSON line then exits. We should receive it.
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case ev, ok := <-ch:
		if !ok {
			// Channel closed before delivering; that's still valid if the process was too fast.
			return
		}
		if ev.Type != "result" {
			t.Errorf("event Type = %q, want %q", ev.Type, "result")
		}
	case <-timer.C:
		t.Error("timed out waiting for event from StreamEvents")
	}
}

func TestStreamEvents_NotFound(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	_, err := h.Bridge.StreamEvents(ctx, "nonexistent-session")
	if err == nil {
		t.Fatal("expected error for nonexistent session, got nil")
	}
}
