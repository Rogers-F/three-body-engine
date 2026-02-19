package team

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
)

// TimeoutAction records a timeout action taken against a worker.
type TimeoutAction struct {
	WorkerID string
	Type     string // "soft" or "hard"
}

// SupervisorConfig holds tunable parameters for the supervisor loop.
type SupervisorConfig struct {
	CheckIntervalSec int
	HeartbeatMaxAge  int
}

// Supervisor monitors worker heartbeats and handles timeouts.
type Supervisor struct {
	DB            *sql.DB
	WorkerRepo    *store.WorkerRepo
	AuditRepo     *store.AuditRepo
	WorkerManager *WorkerManager
	Config        SupervisorConfig
	stopCh        chan struct{}
	stopOnce      sync.Once
}

// NewSupervisor creates a Supervisor with sensible defaults for zero-value config fields.
func NewSupervisor(db *sql.DB, wm *WorkerManager, cfg SupervisorConfig) *Supervisor {
	if cfg.CheckIntervalSec == 0 {
		cfg.CheckIntervalSec = 10
	}
	if cfg.HeartbeatMaxAge == 0 {
		cfg.HeartbeatMaxAge = 30
	}
	return &Supervisor{
		DB:            db,
		WorkerRepo:    wm.WorkerRepo,
		AuditRepo:     wm.AuditRepo,
		WorkerManager: wm,
		Config:        cfg,
		stopCh:        make(chan struct{}),
	}
}

// Heartbeat updates the heartbeat timestamp for a worker.
func (s *Supervisor) Heartbeat(ctx context.Context, workerID string) error {
	return s.WorkerRepo.UpdateHeartbeat(ctx, s.DB, workerID, time.Now().Unix())
}

// CheckTimeouts inspects all active workers for a task and returns actions for any that
// have exceeded their soft or hard timeout thresholds.
func (s *Supervisor) CheckTimeouts(ctx context.Context, taskID string, nowUnix int64) ([]TimeoutAction, error) {
	workers, err := s.WorkerRepo.ListActive(ctx, s.DB, taskID)
	if err != nil {
		return nil, fmt.Errorf("list active workers: %w", err)
	}

	var actions []TimeoutAction
	for _, w := range workers {
		age := nowUnix - w.LastHeartbeat

		if w.HardTimeoutSec > 0 && age > int64(w.HardTimeoutSec) {
			_ = s.WorkerManager.UpdateState(ctx, w.WorkerID, domain.WorkerHardTimeout)
			_, _ = s.WorkerManager.Replace(ctx, w.WorkerID)
			actions = append(actions, TimeoutAction{WorkerID: w.WorkerID, Type: "hard"})

			now := time.Now()
			_ = s.AuditRepo.Record(ctx, s.DB, domain.AuditRecord{
				ID:        fmt.Sprintf("aud-%d", now.UnixNano()),
				TaskID:    w.TaskID,
				Category:  "supervisor",
				Actor:     "system",
				Action:    "hard_timeout",
				Severity:  "warning",
				CreatedAt: now.Unix(),
			})
		} else if w.SoftTimeoutSec > 0 && age > int64(w.SoftTimeoutSec) {
			_ = s.WorkerManager.UpdateState(ctx, w.WorkerID, domain.WorkerSoftTimeout)
			actions = append(actions, TimeoutAction{WorkerID: w.WorkerID, Type: "soft"})

			now := time.Now()
			_ = s.AuditRepo.Record(ctx, s.DB, domain.AuditRecord{
				ID:        fmt.Sprintf("aud-%d", now.UnixNano()),
				TaskID:    w.TaskID,
				Category:  "supervisor",
				Actor:     "system",
				Action:    "soft_timeout",
				Severity:  "warning",
				CreatedAt: now.Unix(),
			})
		}
	}
	return actions, nil
}

// StartMonitoring spawns a goroutine that periodically checks for worker timeouts.
func (s *Supervisor) StartMonitoring(ctx context.Context, taskID string) {
	ticker := time.NewTicker(time.Duration(s.Config.CheckIntervalSec) * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-s.stopCh:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = s.CheckTimeouts(ctx, taskID, time.Now().Unix())
			}
		}
	}()
}

// StopMonitoring signals the monitoring goroutine to stop. Safe to call multiple times.
func (s *Supervisor) StopMonitoring() {
	s.stopOnce.Do(func() { close(s.stopCh) })
}
