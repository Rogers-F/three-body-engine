package team

import (
	"context"
	"database/sql"

	"github.com/anthropics/three-body-engine/internal/domain"
	"github.com/anthropics/three-body-engine/internal/store"
)

// ConflictType classifies the kind of file conflict between intents.
type ConflictType string

const (
	ConflictOverlap ConflictType = "overlap"
	ConflictDelete  ConflictType = "delete"
	ConflictCreate  ConflictType = "create"
)

// FileConflict describes a conflict between two intents on the same file.
type FileConflict struct {
	File    string
	IntentA domain.Intent
	IntentB domain.Intent
	Type    ConflictType
}

// ConflictDetector finds and classifies conflicts between active intents.
type ConflictDetector struct {
	IntentRepo *store.IntentRepo
	DB         *sql.DB
}

// Detect scans all pending and running intents for a task and returns any file conflicts.
func (d *ConflictDetector) Detect(ctx context.Context, taskID string) ([]FileConflict, error) {
	pending, err := d.IntentRepo.ListByTaskStatus(ctx, d.DB, taskID, "pending")
	if err != nil {
		return nil, err
	}
	running, err := d.IntentRepo.ListByTaskStatus(ctx, d.DB, taskID, "running")
	if err != nil {
		return nil, err
	}

	all := append(pending, running...)

	byFile := make(map[string][]domain.Intent)
	for _, intent := range all {
		byFile[intent.TargetFile] = append(byFile[intent.TargetFile], intent)
	}

	var conflicts []FileConflict
	for _, intents := range byFile {
		if len(intents) < 2 {
			continue
		}
		for i := 0; i < len(intents); i++ {
			for j := i + 1; j < len(intents); j++ {
				if c := d.DetectBetween(intents[i], intents[j]); c != nil {
					conflicts = append(conflicts, *c)
				}
			}
		}
	}
	return conflicts, nil
}

// DetectBetween checks two intents for a conflict.
// Returns nil if the intents target different files.
func (d *ConflictDetector) DetectBetween(a, b domain.Intent) *FileConflict {
	if a.TargetFile != b.TargetFile {
		return nil
	}

	var ctype ConflictType
	switch {
	case a.Operation == "delete" || b.Operation == "delete":
		ctype = ConflictDelete
	case a.Operation == "create" && b.Operation == "create":
		ctype = ConflictCreate
	default:
		ctype = ConflictOverlap
	}

	return &FileConflict{
		File:    a.TargetFile,
		IntentA: a,
		IntentB: b,
		Type:    ctype,
	}
}

// Resolve attempts to resolve a file conflict. In MVP this always returns an error.
func (d *ConflictDetector) Resolve(ctx context.Context, conflict FileConflict) error {
	return domain.ErrIntentConflict
}
