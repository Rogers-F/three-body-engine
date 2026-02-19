package team

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
)

func newSupervisorTestDB(t *testing.T) (*Supervisor, *WorkerManager) {
	t.Helper()
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	mgr := NewWorkerManager(db, 10)
	sup := NewSupervisor(db, mgr, SupervisorConfig{
		CheckIntervalSec: 1,
		HeartbeatMaxAge:  30,
	})
	return sup, mgr
}

func TestNewSupervisor_Defaults(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	mgr := NewWorkerManager(db, 4)
	sup := NewSupervisor(db, mgr, SupervisorConfig{})

	if sup.Config.CheckIntervalSec != 10 {
		t.Errorf("CheckIntervalSec = %d, want 10", sup.Config.CheckIntervalSec)
	}
	if sup.Config.HeartbeatMaxAge != 30 {
		t.Errorf("HeartbeatMaxAge = %d, want 30", sup.Config.HeartbeatMaxAge)
	}
}

func TestHeartbeat_Success(t *testing.T) {
	sup, mgr := newSupervisorTestDB(t)
	ctx := context.Background()

	w, err := mgr.Spawn(ctx, domain.WorkerSpec{
		TaskID:         "task-1",
		Phase:          domain.PhaseC,
		Role:           "coder",
		FileOwnership:  []string{"main.go"},
		SoftTimeoutSec: 300,
		HardTimeoutSec: 600,
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	if err := sup.Heartbeat(ctx, w.WorkerID); err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}

	got, err := sup.WorkerRepo.GetByID(ctx, sup.DB, w.WorkerID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.LastHeartbeat < time.Now().Unix()-2 {
		t.Error("expected LastHeartbeat to be recent")
	}
}

func TestHeartbeat_WorkerNotFound(t *testing.T) {
	sup, _ := newSupervisorTestDB(t)
	ctx := context.Background()

	err := sup.Heartbeat(ctx, "nonexistent-worker")
	if err != domain.ErrWorkerNotFound {
		t.Errorf("expected ErrWorkerNotFound, got %v", err)
	}
}

func TestCheckTimeouts_NoTimeouts(t *testing.T) {
	sup, mgr := newSupervisorTestDB(t)
	ctx := context.Background()

	_, err := mgr.Spawn(ctx, domain.WorkerSpec{
		TaskID:         "task-1",
		Phase:          domain.PhaseC,
		Role:           "coder",
		FileOwnership:  []string{"main.go"},
		SoftTimeoutSec: 300,
		HardTimeoutSec: 600,
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	actions, err := sup.CheckTimeouts(ctx, "task-1", time.Now().Unix())
	if err != nil {
		t.Fatalf("CheckTimeouts: %v", err)
	}
	if len(actions) != 0 {
		t.Errorf("expected 0 actions, got %d", len(actions))
	}
}

func TestCheckTimeouts_SoftTimeout(t *testing.T) {
	sup, mgr := newSupervisorTestDB(t)
	ctx := context.Background()

	w, err := mgr.Spawn(ctx, domain.WorkerSpec{
		TaskID:         "task-1",
		Phase:          domain.PhaseC,
		Role:           "coder",
		FileOwnership:  []string{"main.go"},
		SoftTimeoutSec: 10,
		HardTimeoutSec: 600,
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	// Simulate time passing beyond soft timeout.
	futureTime := w.LastHeartbeat + 15
	actions, err := sup.CheckTimeouts(ctx, "task-1", futureTime)
	if err != nil {
		t.Fatalf("CheckTimeouts: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Type != "soft" {
		t.Errorf("Type = %q, want %q", actions[0].Type, "soft")
	}
	if actions[0].WorkerID != w.WorkerID {
		t.Errorf("WorkerID = %q, want %q", actions[0].WorkerID, w.WorkerID)
	}
}

func TestCheckTimeouts_HardTimeout(t *testing.T) {
	sup, mgr := newSupervisorTestDB(t)
	ctx := context.Background()

	w, err := mgr.Spawn(ctx, domain.WorkerSpec{
		TaskID:         "task-1",
		Phase:          domain.PhaseC,
		Role:           "coder",
		FileOwnership:  []string{"main.go"},
		SoftTimeoutSec: 10,
		HardTimeoutSec: 30,
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	// Simulate time passing beyond hard timeout.
	futureTime := w.LastHeartbeat + 35
	actions, err := sup.CheckTimeouts(ctx, "task-1", futureTime)
	if err != nil {
		t.Fatalf("CheckTimeouts: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Type != "hard" {
		t.Errorf("Type = %q, want %q", actions[0].Type, "hard")
	}
}

func TestCheckTimeouts_MixedTimeouts(t *testing.T) {
	sup, mgr := newSupervisorTestDB(t)
	ctx := context.Background()

	w1, err := mgr.Spawn(ctx, domain.WorkerSpec{
		TaskID:         "task-1",
		Phase:          domain.PhaseC,
		Role:           "coder",
		FileOwnership:  []string{"a.go"},
		SoftTimeoutSec: 10,
		HardTimeoutSec: 600,
	})
	if err != nil {
		t.Fatalf("Spawn w1: %v", err)
	}

	w2, err := mgr.Spawn(ctx, domain.WorkerSpec{
		TaskID:         "task-1",
		Phase:          domain.PhaseC,
		Role:           "reviewer",
		FileOwnership:  []string{"b.go"},
		SoftTimeoutSec: 10,
		HardTimeoutSec: 20,
	})
	if err != nil {
		t.Fatalf("Spawn w2: %v", err)
	}

	// Use a time that exceeds w1's soft but only w2's hard.
	futureTime := w1.LastHeartbeat + 25
	actions, err := sup.CheckTimeouts(ctx, "task-1", futureTime)
	if err != nil {
		t.Fatalf("CheckTimeouts: %v", err)
	}

	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}

	actionMap := make(map[string]string)
	for _, a := range actions {
		actionMap[a.WorkerID] = a.Type
	}

	if actionMap[w1.WorkerID] != "soft" {
		t.Errorf("w1 action = %q, want %q", actionMap[w1.WorkerID], "soft")
	}
	if actionMap[w2.WorkerID] != "hard" {
		t.Errorf("w2 action = %q, want %q", actionMap[w2.WorkerID], "hard")
	}
}

func TestStartStopMonitoring(t *testing.T) {
	sup, _ := newSupervisorTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sup.StartMonitoring(ctx, "task-1")

	// Let the ticker fire at least once.
	time.Sleep(1500 * time.Millisecond)

	sup.StopMonitoring()
	// No panic or hang means success.
}
