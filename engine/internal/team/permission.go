package team

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
)

// PermissionBroker manages capability sheets and permission checks.
type PermissionBroker struct {
	AuditRepo *store.AuditRepo
	DB        *sql.DB
}

// NewPermissionBroker creates a PermissionBroker with default repos.
func NewPermissionBroker(db *sql.DB) *PermissionBroker {
	return &PermissionBroker{
		AuditRepo: &store.AuditRepo{},
		DB:        db,
	}
}

// defaultDeniedPatterns are file patterns that are always denied.
var defaultDeniedPatterns = []string{".env", "*.key", ".git/*"}

// BuildCapabilitySheet creates a capability sheet with the given allowed paths and commands,
// plus default denied patterns.
func (p *PermissionBroker) BuildCapabilitySheet(taskID string, paths, commands []string) *domain.CapabilitySheet {
	return &domain.CapabilitySheet{
		TaskID:          taskID,
		AllowedPaths:    paths,
		AllowedCommands: commands,
		DeniedPatterns:  defaultDeniedPatterns,
		CreatedAtUnix:   time.Now().Unix(),
	}
}

// CheckPermission verifies whether a path and command are allowed by the capability sheet.
// Returns (true, nil) if allowed, (false, nil) if denied. Denied attempts are audited.
func (p *PermissionBroker) CheckPermission(ctx context.Context, sheet *domain.CapabilitySheet, path, command string) (bool, error) {
	for _, pattern := range sheet.DeniedPatterns {
		matched, err := matchPattern(pattern, path)
		if err != nil {
			return false, fmt.Errorf("match denied pattern %q: %w", pattern, err)
		}
		if matched {
			p.auditDenial(ctx, sheet.TaskID, path, command, "denied by pattern: "+pattern)
			return false, nil
		}
	}

	pathAllowed := false
	for _, allowed := range sheet.AllowedPaths {
		if strings.HasPrefix(path, allowed) {
			pathAllowed = true
			break
		}
	}
	if !pathAllowed {
		p.auditDenial(ctx, sheet.TaskID, path, command, "path not in allowed list")
		return false, nil
	}

	cmdAllowed := false
	for _, allowed := range sheet.AllowedCommands {
		if command == allowed {
			cmdAllowed = true
			break
		}
	}
	if !cmdAllowed {
		p.auditDenial(ctx, sheet.TaskID, path, command, "command not in allowed list")
		return false, nil
	}

	return true, nil
}

func (p *PermissionBroker) auditDenial(ctx context.Context, taskID, path, command, reason string) {
	now := time.Now()
	_ = p.AuditRepo.Record(ctx, p.DB, domain.AuditRecord{
		ID:           fmt.Sprintf("aud-perm-%d", now.UnixNano()),
		TaskID:       taskID,
		Category:     "permission",
		Actor:        "system",
		Action:       "permission_denied",
		RequestJSON:  fmt.Sprintf(`{"path":%q,"command":%q}`, path, command),
		DecisionJSON: fmt.Sprintf(`{"reason":%q}`, reason),
		Severity:     "warning",
		CreatedAt:    now.Unix(),
	})
}

// matchPattern checks if a path matches a denied pattern.
// Supports exact match (e.g., ".env"), glob match via filepath.Match, and prefix match for directory patterns.
func matchPattern(pattern, path string) (bool, error) {
	// Exact match
	if path == pattern {
		return true, nil
	}

	// Base name match (e.g., ".env" matches "some/dir/.env")
	base := filepath.Base(path)
	if base == pattern {
		return true, nil
	}

	// Glob match on the full path
	matched, err := filepath.Match(pattern, path)
	if err != nil {
		return false, err
	}
	if matched {
		return true, nil
	}

	// Glob match on the base name
	matched, err = filepath.Match(pattern, base)
	if err != nil {
		return false, err
	}
	return matched, nil
}
