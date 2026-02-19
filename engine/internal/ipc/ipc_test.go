package ipc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/guard"
	"github.com/anthropics/three-body-engine/internal/store"
	"github.com/anthropics/three-body-engine/internal/team"
	"github.com/anthropics/three-body-engine/internal/workflow"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := store.NewDB(dbPath)
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	gov := workflow.NewBudgetGovernor(db)
	broker := team.NewPermissionBroker(db)
	g := guard.NewGuard(db, gov, broker, guard.GuardConfig{
		MaxRounds:          10,
		RateLimitPerMinute: 1000,
	})

	engine := workflow.NewEngine(db)

	return &Handler{
		Engine:        engine,
		Guard:         g,
		DB:            db,
		EventRepo:     &store.EventRepo{},
		WorkerRepo:    &store.WorkerRepo{},
		ScoreCardRepo: &store.ScoreCardRepo{},
		CostDeltaRepo: &store.CostDeltaRepo{},
		TaskRepo:      &store.TaskRepo{},
	}
}

func TestCreateFlow_Success(t *testing.T) {
	h := newTestHandler(t)
	body := `{"task_id":"t1","budget_cap_usd":10.0}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flow", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.CreateFlow(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var state domain.FlowState
	json.NewDecoder(w.Body).Decode(&state)
	if state.TaskID != "t1" {
		t.Errorf("expected task_id=t1, got %s", state.TaskID)
	}
	if state.CurrentPhase != domain.PhaseA {
		t.Errorf("expected phase A, got %s", state.CurrentPhase)
	}
}

func TestCreateFlow_InvalidBody(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flow", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()

	h.CreateFlow(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetFlow_Success(t *testing.T) {
	h := newTestHandler(t)
	ctx := context.Background()
	h.Engine.StartFlow(ctx, "t1", 10.0)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/flow/t1", nil)
	req.SetPathValue("taskID", "t1")
	w := httptest.NewRecorder()

	h.GetFlow(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetFlow_NotFound(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/flow/nonexistent", nil)
	req.SetPathValue("taskID", "nonexistent")
	w := httptest.NewRecorder()

	h.GetFlow(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAdvanceFlow_Success(t *testing.T) {
	h := newTestHandler(t)
	ctx := context.Background()
	h.Engine.StartFlow(ctx, "t1", 10.0)

	body := `{"action":"advance","actor":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flow/t1/advance", bytes.NewBufferString(body))
	req.SetPathValue("taskID", "t1")
	w := httptest.NewRecorder()

	h.AdvanceFlow(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	state, _ := h.Engine.GetState(ctx, "t1")
	if state.CurrentPhase != domain.PhaseB {
		t.Errorf("expected phase B, got %s", state.CurrentPhase)
	}
}

func TestAdvanceFlow_InvalidAction(t *testing.T) {
	h := newTestHandler(t)
	ctx := context.Background()
	h.Engine.StartFlow(ctx, "t1", 10.0)

	body := `{"action":"","actor":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flow/t1/advance", bytes.NewBufferString(body))
	req.SetPathValue("taskID", "t1")
	w := httptest.NewRecorder()

	h.AdvanceFlow(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListWorkers_Empty(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/flow/t1/workers", nil)
	req.SetPathValue("taskID", "t1")
	w := httptest.NewRecorder()

	h.ListWorkers(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var workers []domain.WorkerRef
	json.NewDecoder(w.Body).Decode(&workers)
	if len(workers) != 0 {
		t.Errorf("expected 0 workers, got %d", len(workers))
	}
}

func TestListEvents_ReturnsEvents(t *testing.T) {
	h := newTestHandler(t)
	ctx := context.Background()
	h.Engine.StartFlow(ctx, "t1", 10.0)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/flow/t1/events", nil)
	req.SetPathValue("taskID", "t1")
	w := httptest.NewRecorder()

	h.ListEvents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var events []domain.WorkflowEvent
	json.NewDecoder(w.Body).Decode(&events)
	if len(events) == 0 {
		t.Error("expected at least 1 event (flow_started)")
	}
}

func TestGetCost_ReturnsSummary(t *testing.T) {
	h := newTestHandler(t)
	ctx := context.Background()
	h.Engine.StartFlow(ctx, "t1", 10.0)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/flow/t1/cost", nil)
	req.SetPathValue("taskID", "t1")
	w := httptest.NewRecorder()

	h.GetCost(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var summary CostSummary
	json.NewDecoder(w.Body).Decode(&summary)
	if summary.BudgetCapUSD != 10.0 {
		t.Errorf("expected budget_cap=10.0, got %f", summary.BudgetCapUSD)
	}
}

func TestStreamEvents_SSE_FirstBatch(t *testing.T) {
	h := newTestHandler(t)
	ctx := context.Background()
	h.Engine.StartFlow(ctx, "t1", 10.0)

	// Use a cancellable context so the SSE handler returns.
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/flow/t1/events/stream", nil).WithContext(ctx)
	req.SetPathValue("taskID", "t1")
	w := httptest.NewRecorder()

	h.StreamEvents(w, req)

	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %s", ct)
	}
	if w.Body.Len() == 0 {
		t.Error("expected SSE data in body")
	}
}

func TestCORSHeaders(t *testing.T) {
	h := newTestHandler(t)
	srv := NewServer(h, ":0")

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/flow/t1", nil)
	w := httptest.NewRecorder()

	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS origin *")
	}
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS, got %d", w.Code)
	}
}

func TestListReviews_Empty(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/flow/t1/reviews", nil)
	req.SetPathValue("taskID", "t1")
	w := httptest.NewRecorder()

	h.ListReviews(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var cards []domain.ScoreCard
	json.NewDecoder(w.Body).Decode(&cards)
	if len(cards) != 0 {
		t.Errorf("expected 0 cards, got %d", len(cards))
	}
}

