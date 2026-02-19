package mcp

import (
	"context"
	"encoding/json"
	"os/exec"
	"runtime"
	"sync"
	"testing"

	"github.com/anthropics/three-body-engine/internal/domain"
)

// ---------------------------------------------------------------------------
// ProviderRegistry tests
// ---------------------------------------------------------------------------

func TestProviderRegistry_RegisterAndGet(t *testing.T) {
	reg := NewProviderRegistry()
	spec := ProviderSpec{
		Name:    domain.ProviderClaude,
		Command: "echo",
		Args:    []string{"hello"},
		Env:     map[string]string{"KEY": "VAL"},
	}

	if err := reg.Register(spec); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, err := reg.Get(domain.ProviderClaude)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Command != "echo" {
		t.Errorf("Command = %q, want %q", got.Command, "echo")
	}
	if got.Env["KEY"] != "VAL" {
		t.Errorf("Env[KEY] = %q, want %q", got.Env["KEY"], "VAL")
	}
}

func TestProviderRegistry_RegisterDuplicate(t *testing.T) {
	reg := NewProviderRegistry()
	spec := ProviderSpec{Name: domain.ProviderCodex, Command: "echo"}

	if err := reg.Register(spec); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	err := reg.Register(spec)
	if err == nil {
		t.Fatal("expected error on duplicate register, got nil")
	}
}

func TestProviderRegistry_GetUnknown(t *testing.T) {
	reg := NewProviderRegistry()
	_, err := reg.Get(domain.Provider("nonexistent"))
	if err != domain.ErrProviderUnavailable {
		t.Errorf("err = %v, want ErrProviderUnavailable", err)
	}
}

func TestProviderRegistry_List(t *testing.T) {
	reg := NewProviderRegistry()
	providers := []ProviderSpec{
		{Name: domain.ProviderGemini, Command: "echo"},
		{Name: domain.ProviderClaude, Command: "echo"},
		{Name: domain.ProviderCodex, Command: "echo"},
	}
	for _, p := range providers {
		if err := reg.Register(p); err != nil {
			t.Fatalf("Register %s: %v", p.Name, err)
		}
	}

	list := reg.List()
	if len(list) != 3 {
		t.Fatalf("List len = %d, want 3", len(list))
	}
	// List returns sorted order.
	if list[0] != domain.ProviderClaude {
		t.Errorf("list[0] = %q, want %q", list[0], domain.ProviderClaude)
	}
	if list[1] != domain.ProviderCodex {
		t.Errorf("list[1] = %q, want %q", list[1], domain.ProviderCodex)
	}
	if list[2] != domain.ProviderGemini {
		t.Errorf("list[2] = %q, want %q", list[2], domain.ProviderGemini)
	}
}

// ---------------------------------------------------------------------------
// SessionManager tests
// ---------------------------------------------------------------------------

// echoCommand returns the OS-appropriate command that echoes a JSON line and exits.
func echoCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", `echo {"type":"result","data":"ok"}`}
	}
	return "sh", []string{"-c", `echo '{"type":"result","data":"ok"}'`}
}

func newTestRegistry(t *testing.T) *ProviderRegistry {
	t.Helper()
	reg := NewProviderRegistry()
	cmd, args := echoCommand()
	if err := reg.Register(ProviderSpec{
		Name:    domain.ProviderClaude,
		Command: cmd,
		Args:    args,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	return reg
}

func TestSessionManager_CreateAndGet(t *testing.T) {
	reg := newTestRegistry(t)
	mgr := NewSessionManager(reg)
	defer mgr.StopAll()

	ctx := context.Background()
	cfg := domain.SessionConfig{TaskID: "task-1", Role: "coder", Workspace: t.TempDir()}

	id, err := mgr.Create(ctx, domain.ProviderClaude, cfg)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty session ID")
	}

	sess, err := mgr.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if sess.Provider != domain.ProviderClaude {
		t.Errorf("Provider = %q, want %q", sess.Provider, domain.ProviderClaude)
	}
}

func TestSessionManager_CreateUnknownProvider(t *testing.T) {
	reg := NewProviderRegistry()
	mgr := NewSessionManager(reg)

	ctx := context.Background()
	_, err := mgr.Create(ctx, domain.Provider("unknown"), domain.SessionConfig{})
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
}

func TestSessionManager_Stop(t *testing.T) {
	reg := newTestRegistry(t)
	mgr := NewSessionManager(reg)

	ctx := context.Background()
	id, err := mgr.Create(ctx, domain.ProviderClaude, domain.SessionConfig{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := mgr.Stop(id); err != nil {
		// On Windows, killing an already-exited process returns an error. That's OK.
		t.Logf("Stop returned (may be expected on fast exit): %v", err)
	}

	_, err = mgr.Get(id)
	if err != domain.ErrSessionNotFound {
		t.Errorf("Get after stop: err = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionManager_StopNotFound(t *testing.T) {
	reg := NewProviderRegistry()
	mgr := NewSessionManager(reg)

	err := mgr.Stop("nonexistent")
	if err != domain.ErrSessionNotFound {
		t.Errorf("err = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionManager_StopAll(t *testing.T) {
	reg := newTestRegistry(t)
	mgr := NewSessionManager(reg)

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := mgr.Create(ctx, domain.ProviderClaude, domain.SessionConfig{Workspace: t.TempDir()}); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}

	mgr.StopAll()

	mgr.mu.RLock()
	remaining := len(mgr.sessions)
	mgr.mu.RUnlock()

	if remaining != 0 {
		t.Errorf("sessions remaining after StopAll = %d, want 0", remaining)
	}
}

// ---------------------------------------------------------------------------
// Session unit tests
// ---------------------------------------------------------------------------

func TestParseEvent(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantTyp string
	}{
		{"valid_result", `{"type":"result","data":"ok"}`, false, "result"},
		{"valid_cost", `{"type":"cost","tokens":100}`, false, "cost"},
		{"missing_type", `{"data":"ok"}`, true, ""},
		{"invalid_json", `not json`, true, ""},
		{"empty_type", `{"type":""}`, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev, err := parseEvent([]byte(tt.input), domain.ProviderClaude, "ses-test")
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ev.Type != tt.wantTyp {
				t.Errorf("Type = %q, want %q", ev.Type, tt.wantTyp)
			}
			if ev.Provider != domain.ProviderClaude {
				t.Errorf("Provider = %q, want %q", ev.Provider, domain.ProviderClaude)
			}
			if ev.SessionID != "ses-test" {
				t.Errorf("SessionID = %q, want %q", ev.SessionID, "ses-test")
			}
		})
	}
}

func TestSession_StopTerminatesProcess(t *testing.T) {
	// Use a long-running command so we can actually kill it.
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", "ping -n 60 127.0.0.1 >nul")
	} else {
		cmd = exec.Command("sleep", "60")
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}

	sess := &Session{
		ID:       "ses-kill-test",
		Provider: domain.ProviderClaude,
		cmd:      cmd,
		stdout:   stdout,
		events:   make(chan domain.NormalizedEvent, eventChannelBuffer),
		done:     make(chan struct{}),
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sess.readStdout()
	}()

	if err := sess.Stop(); err != nil {
		t.Logf("Stop returned (expected on kill): %v", err)
	}

	// Verify done channel is closed.
	select {
	case <-sess.Done():
		// expected
	default:
		// readStdout goroutine might need a moment to finish.
		wg.Wait()
		select {
		case <-sess.Done():
		default:
			t.Error("done channel not closed after Stop")
		}
	}
}

func TestParseEvent_PayloadCopy(t *testing.T) {
	// Verify that the returned Payload is an independent copy.
	raw := []byte(`{"type":"test"}`)
	ev, err := parseEvent(raw, domain.ProviderClaude, "ses-1")
	if err != nil {
		t.Fatalf("parseEvent: %v", err)
	}

	// Mutate original.
	raw[0] = 'X'
	var m map[string]interface{}
	if err := json.Unmarshal(ev.Payload, &m); err != nil {
		t.Fatalf("payload corrupted: %v", err)
	}
	if m["type"] != "test" {
		t.Error("payload was not an independent copy")
	}
}
