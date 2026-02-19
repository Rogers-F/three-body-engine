// Package ipc provides the HTTP API for the Three-Body Engine.
package ipc

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/anthropics/three-body-engine/internal/bridge"
	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/guard"
	"github.com/anthropics/three-body-engine/internal/store"
	"github.com/anthropics/three-body-engine/internal/workflow"
)

// Handler holds all dependencies for the HTTP handlers.
type Handler struct {
	Engine        *workflow.Engine
	Bridge        *bridge.Bridge
	Guard         *guard.Guard
	DB            *sql.DB
	EventRepo     *store.EventRepo
	WorkerRepo    *store.WorkerRepo
	ScoreCardRepo *store.ScoreCardRepo
	CostDeltaRepo *store.CostDeltaRepo
	TaskRepo      *store.TaskRepo
}

// CreateFlowRequest is the body for POST /api/v1/flow.
type CreateFlowRequest struct {
	TaskID       string  `json:"task_id"`
	BudgetCapUSD float64 `json:"budget_cap_usd"`
}

// AdvanceRequest is the body for POST /api/v1/flow/{taskID}/advance.
type AdvanceRequest struct {
	Action string `json:"action"`
	Actor  string `json:"actor"`
}

// CostSummary is the response for GET /api/v1/flow/{taskID}/cost.
type CostSummary struct {
	BudgetUsedUSD float64           `json:"budget_used_usd"`
	BudgetCapUSD  float64           `json:"budget_cap_usd"`
	CostAction    domain.CostAction `json:"cost_action"`
	Deltas        []domain.CostDelta `json:"deltas"`
}

// APIError is a structured error response.
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// GetFlow handles GET /api/v1/flow/{taskID}.
func (h *Handler) GetFlow(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	state, err := h.Engine.GetState(r.Context(), taskID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// CreateFlow handles POST /api/v1/flow.
func (h *Handler) CreateFlow(w http.ResponseWriter, r *http.Request) {
	var req CreateFlowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Code: 400, Message: "invalid request body"})
		return
	}
	if req.TaskID == "" {
		writeJSON(w, http.StatusBadRequest, APIError{Code: 400, Message: "task_id is required"})
		return
	}

	if err := h.Engine.StartFlow(r.Context(), req.TaskID, req.BudgetCapUSD); err != nil {
		writeError(w, err)
		return
	}

	state, err := h.Engine.GetState(r.Context(), req.TaskID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, state)
}

// AdvanceFlow handles POST /api/v1/flow/{taskID}/advance.
func (h *Handler) AdvanceFlow(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	var req AdvanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Code: 400, Message: "invalid request body"})
		return
	}
	if req.Action == "" {
		writeJSON(w, http.StatusBadRequest, APIError{Code: 400, Message: "action is required"})
		return
	}

	trigger := domain.TransitionTrigger{
		Action: req.Action,
		Actor:  req.Actor,
	}
	if err := h.Engine.Advance(r.Context(), taskID, trigger); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListWorkers handles GET /api/v1/flow/{taskID}/workers.
func (h *Handler) ListWorkers(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	workers, err := h.WorkerRepo.ListByTask(r.Context(), h.DB, taskID)
	if err != nil {
		writeError(w, err)
		return
	}
	if workers == nil {
		workers = []*domain.WorkerRef{}
	}
	writeJSON(w, http.StatusOK, workers)
}

// ListEvents handles GET /api/v1/flow/{taskID}/events?since_seq=N.
func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	sinceSeq := int64(0)
	if s := r.URL.Query().Get("since_seq"); s != "" {
		parsed, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			sinceSeq = parsed
		}
	}

	events, err := h.EventRepo.ListByTask(r.Context(), h.DB, taskID, sinceSeq)
	if err != nil {
		writeError(w, err)
		return
	}
	if events == nil {
		events = []domain.WorkflowEvent{}
	}
	writeJSON(w, http.StatusOK, events)
}

// ListReviews handles GET /api/v1/flow/{taskID}/reviews.
func (h *Handler) ListReviews(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	cards, err := h.ScoreCardRepo.ListByTask(r.Context(), h.DB, taskID)
	if err != nil {
		writeError(w, err)
		return
	}
	if cards == nil {
		cards = []domain.ScoreCard{}
	}
	writeJSON(w, http.StatusOK, cards)
}

// GetCost handles GET /api/v1/flow/{taskID}/cost.
func (h *Handler) GetCost(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	state, err := h.TaskRepo.GetByID(r.Context(), h.DB, taskID)
	if err != nil {
		writeError(w, err)
		return
	}

	deltas, err := h.CostDeltaRepo.ListByTask(r.Context(), h.DB, taskID)
	if err != nil {
		writeError(w, err)
		return
	}
	if deltas == nil {
		deltas = []domain.CostDelta{}
	}

	action, _ := h.Guard.CheckBudget(r.Context(), taskID)

	summary := CostSummary{
		BudgetUsedUSD: state.BudgetUsedUSD,
		BudgetCapUSD:  state.BudgetCapUSD,
		CostAction:    action,
		Deltas:        deltas,
	}
	writeJSON(w, http.StatusOK, summary)
}

// StreamEvents handles GET /api/v1/flow/{taskID}/events/stream (SSE).
func (h *Handler) StreamEvents(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, APIError{Code: 500, Message: "streaming not supported"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Send initial batch of events.
	events, err := h.EventRepo.ListByTask(r.Context(), h.DB, taskID, 0)
	if err != nil {
		writeSSEError(w, flusher, err)
		return
	}
	for _, ev := range events {
		writeSSEEvent(w, flusher, ev)
	}

	// Poll for new events.
	lastSeq := int64(0)
	if len(events) > 0 {
		lastSeq = events[len(events)-1].SeqNo
	}

	ctx := r.Context()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			newEvents, err := h.EventRepo.ListByTask(ctx, h.DB, taskID, lastSeq)
			if err != nil {
				return
			}
			for _, ev := range newEvents {
				writeSSEEvent(w, flusher, ev)
				lastSeq = ev.SeqNo
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	if engErr, ok := err.(*domain.EngineError); ok {
		status := http.StatusInternalServerError
		switch engErr.Code {
		case domain.ErrFlowNotFound.Code:
			status = http.StatusNotFound
		case domain.ErrDuplicateTask.Code:
			status = http.StatusConflict
		case domain.ErrBudgetExceeded.Code:
			status = http.StatusForbidden
		case domain.ErrRateLimitExceeded.Code:
			status = http.StatusTooManyRequests
		case domain.ErrInvalidTransition.Code, domain.ErrPhaseGateFailed.Code:
			status = http.StatusUnprocessableEntity
		}
		writeJSON(w, status, APIError{Code: engErr.Code, Message: engErr.Message})
		return
	}
	writeJSON(w, http.StatusInternalServerError, APIError{Code: -1, Message: err.Error()})
}

func writeSSEEvent(w http.ResponseWriter, f http.Flusher, ev domain.WorkflowEvent) {
	data, _ := json.Marshal(ev)
	fmt.Fprintf(w, "data: %s\n\n", data)
	f.Flush()
}

func writeSSEError(w http.ResponseWriter, f http.Flusher, err error) {
	fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
	f.Flush()
}
