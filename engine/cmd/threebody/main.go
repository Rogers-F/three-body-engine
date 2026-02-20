// Package main is the entry point for the Three-Body Engine.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anthropics/three-body-engine/internal/bridge"
	"github.com/anthropics/three-body-engine/internal/config"
	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/guard"
	"github.com/anthropics/three-body-engine/internal/ipc"
	"github.com/anthropics/three-body-engine/internal/mcp"
	"github.com/anthropics/three-body-engine/internal/store"
	"github.com/anthropics/three-body-engine/internal/team"
	"github.com/anthropics/three-body-engine/internal/workflow"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	configPath := flag.String("config", "", "path to configuration JSON file")
	flag.Parse()

	if *showVersion {
		fmt.Printf("threebody %s (commit=%s, built=%s)\n", version, commit, date)
		os.Exit(0)
	}

	// Fallback to environment variable.
	path := *configPath
	if path == "" {
		path = os.Getenv("TB_CONFIG")
	}
	if path == "" {
		fmt.Fprintln(os.Stderr, "usage: threebody --config <path> (or set TB_CONFIG)")
		os.Exit(1)
	}

	cfg, err := config.Load(path)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := store.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	// Wire workflow engine.
	engine := workflow.NewEngine(db)
	gov := workflow.NewBudgetGovernor(db)

	// Wire team management.
	broker := team.NewPermissionBroker(db)
	wm := team.NewWorkerManager(db, cfg.MaxConcurrentWorkers)
	supervisor := team.NewSupervisor(db, wm, team.SupervisorConfig{
		CheckIntervalSec: cfg.CheckIntervalSec,
		HeartbeatMaxAge:  cfg.HeartbeatMaxAge,
	})

	// Wire provider registry.
	registry := mcp.NewProviderRegistry()
	for name, pc := range cfg.Providers {
		if err := registry.Register(mcp.ProviderSpec{
			Name:    domain.Provider(name),
			Command: pc.Command,
			Args:    pc.Args,
			Env:     pc.Env,
		}); err != nil {
			log.Fatalf("register provider %s: %v", name, err)
		}
	}

	// Shared repos.
	costDeltaRepo := &store.CostDeltaRepo{}
	auditRepo := &store.AuditRepo{}
	eventRepo := &store.EventRepo{}
	workerRepo := &store.WorkerRepo{}
	scoreCardRepo := &store.ScoreCardRepo{}
	taskRepo := &store.TaskRepo{}

	// Wire session manager, guard, and bridge.
	sessions := mcp.NewSessionManager(registry)
	g := guard.NewGuard(db, gov, broker, guard.GuardConfig{
		MaxRounds:          cfg.MaxRounds,
		RateLimitPerMinute: cfg.RateLimitPerMinute,
	})

	b := bridge.NewBridge(sessions, g, gov, costDeltaRepo, auditRepo, db)

	// Wire IPC handler.
	handler := &ipc.Handler{
		Engine:        engine,
		Bridge:        b,
		Guard:         g,
		DB:            db,
		EventRepo:     eventRepo,
		WorkerRepo:    workerRepo,
		ScoreCardRepo: scoreCardRepo,
		CostDeltaRepo: costDeltaRepo,
		TaskRepo:      taskRepo,
	}

	srv := ipc.NewServer(handler, cfg.ListenAddr)

	// Graceful shutdown on interrupt.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("shutting down...")

		supervisor.StopMonitoring()
		sessions.StopAll()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("server shutdown: %v", err)
		}
	}()

	log.Printf("three-body engine listening on %s", cfg.ListenAddr)

	// Start supervisor monitoring (runs in background goroutine).
	// Note: monitoring a specific task ID is deferred until flow creation;
	// this is a no-op placeholder that demonstrates supervisor wiring.
	_ = supervisor
	_ = wm

	if err := srv.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
