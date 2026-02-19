package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anthropics/three-body-engine/internal/domain"
)

const eventChannelBuffer = 64

// Session represents a running code agent process communicating via JSON lines on stdout.
type Session struct {
	ID        string
	Provider  domain.Provider
	Config    domain.SessionConfig
	cmd       *exec.Cmd
	stdout    io.ReadCloser
	events    chan domain.NormalizedEvent
	done      chan struct{}
	doneOnce  sync.Once
	startedAt int64
}

// Start launches the provider process and begins reading events from stdout.
func (s *Session) Start(ctx context.Context) error {
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("start session %s: %w", s.ID, err)
	}
	s.startedAt = time.Now().UnixNano()

	go s.readStdout()
	return nil
}

// Stop terminates the provider process. Safe for Windows (uses Process.Kill).
// Wait is called after Kill to reclaim OS resources and avoid zombie processes.
func (s *Session) Stop() error {
	if s.cmd.Process == nil {
		return nil
	}
	err := s.cmd.Process.Kill()
	// Wait reclaims OS process resources. Ignore its error since Kill
	// already signals termination; Wait may return "process already finished".
	_ = s.cmd.Wait()
	s.markDone()
	return err
}

// Events returns a receive-only channel of normalized events from the provider.
func (s *Session) Events() <-chan domain.NormalizedEvent {
	return s.events
}

// Done returns a channel that is closed when the session terminates.
func (s *Session) Done() <-chan struct{} {
	return s.done
}

func (s *Session) markDone() {
	s.doneOnce.Do(func() {
		close(s.done)
	})
}

// readStdout reads JSON lines from the process stdout and publishes NormalizedEvent values.
func (s *Session) readStdout() {
	defer s.markDone()
	defer close(s.events)

	scanner := bufio.NewScanner(s.stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		ev, err := parseEvent(line, s.Provider, s.ID)
		if err != nil {
			continue
		}
		s.events <- ev
	}
}

// parseEvent converts a JSON line into a NormalizedEvent.
func parseEvent(line []byte, provider domain.Provider, sessionID string) (domain.NormalizedEvent, error) {
	var raw struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(line, &raw); err != nil {
		return domain.NormalizedEvent{}, err
	}
	if raw.Type == "" {
		return domain.NormalizedEvent{}, fmt.Errorf("event has no type field")
	}
	return domain.NormalizedEvent{
		Type:      raw.Type,
		Provider:  provider,
		SessionID: sessionID,
		Payload:   append([]byte(nil), line...),
	}, nil
}

// SessionManager creates, tracks, and stops code agent sessions.
type SessionManager struct {
	registry *ProviderRegistry
	mu       sync.RWMutex
	sessions map[string]*Session
	seq      atomic.Int64
}

// NewSessionManager creates a manager backed by the given provider registry.
func NewSessionManager(registry *ProviderRegistry) *SessionManager {
	return &SessionManager{
		registry: registry,
		sessions: make(map[string]*Session),
	}
}

// Create starts a new code agent session for the given provider and config.
func (m *SessionManager) Create(ctx context.Context, provider domain.Provider, cfg domain.SessionConfig) (string, error) {
	spec, err := m.registry.Get(provider)
	if err != nil {
		return "", err
	}

	id := fmt.Sprintf("ses-%s-%d-%d", provider, time.Now().UnixNano(), m.seq.Add(1))
	cmd := exec.CommandContext(ctx, spec.Command, spec.Args...)

	// Merge provider env with session-specific env.
	for k, v := range spec.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe for %s: %w", id, err)
	}

	sess := &Session{
		ID:       id,
		Provider: provider,
		Config:   cfg,
		cmd:      cmd,
		stdout:   stdout,
		events:   make(chan domain.NormalizedEvent, eventChannelBuffer),
		done:     make(chan struct{}),
	}

	if err := sess.Start(ctx); err != nil {
		return "", err
	}

	m.mu.Lock()
	m.sessions[id] = sess
	m.mu.Unlock()

	return id, nil
}

// Get returns a session by ID, or ErrSessionNotFound.
func (m *SessionManager) Get(sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sess, ok := m.sessions[sessionID]
	if !ok {
		return nil, domain.ErrSessionNotFound
	}
	return sess, nil
}

// Stop terminates a session by ID, or returns ErrSessionNotFound.
func (m *SessionManager) Stop(sessionID string) error {
	m.mu.Lock()
	sess, ok := m.sessions[sessionID]
	if !ok {
		m.mu.Unlock()
		return domain.ErrSessionNotFound
	}
	delete(m.sessions, sessionID)
	m.mu.Unlock()

	return sess.Stop()
}

// StopAll terminates every tracked session.
func (m *SessionManager) StopAll() {
	m.mu.Lock()
	sessions := make(map[string]*Session, len(m.sessions))
	for k, v := range m.sessions {
		sessions[k] = v
	}
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()

	for _, sess := range sessions {
		sess.Stop()
	}
}
