// Package main is the entry point for the Three-Body Engine.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
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

	// Resolve config path: --config flag > TB_CONFIG env > auto-discover next to exe.
	path := *configPath
	if path == "" {
		path = os.Getenv("TB_CONFIG")
	}
	if path == "" {
		path = discoverConfig()
	}
	if path == "" {
		fatal("no config found. Place config.json next to the exe, use --config <path>, or set TB_CONFIG.")
	}

	cfg, err := config.Load(path)
	if err != nil {
		fatal(fmt.Sprintf("load config: %v", err))
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

	url := ipc.FormatListenURL(cfg.ListenAddr)
	log.Printf("three-body engine listening on %s", url)

	// Auto-open browser on Windows.
	openBrowser(url)

	_ = supervisor
	_ = wm

	if err := srv.Start(); err != nil && err != http.ErrServerClosed {
		fatal(fmt.Sprintf("server error: %v", err))
	}
}

// discoverConfig looks for config.json next to the executable, then in the cwd.
func discoverConfig() string {
	// Next to executable.
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "config.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	// Current working directory.
	if _, err := os.Stat("config.json"); err == nil {
		return "config.json"
	}
	return ""
}

// fatal prints an error and, on Windows, waits for a keypress so the user can
// read the message when the exe is launched by double-click.
func fatal(msg string) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", msg)
	if runtime.GOOS == "windows" {
		fmt.Fprintln(os.Stderr, "\nPress Enter to exit...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}
	os.Exit(1)
}

// openBrowser opens the URL in the default browser.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
