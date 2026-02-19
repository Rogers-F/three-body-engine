// Package team implements worker lifecycle, compaction, permissions, and context management.
package team

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/three-body-engine/internal/domain"
)

// CompactionValidator validates that all required CompactionSlots are populated.
type CompactionValidator struct{}

// Validate checks that the required semantic slots are present.
// Returns nil if all required slots are valid, or a structured error listing missing slots.
func (v *CompactionValidator) Validate(_ context.Context, slots domain.CompactionSlots) error {
	var missing []string

	if strings.TrimSpace(slots.TaskSpec) == "" {
		missing = append(missing, "TaskSpec")
	}
	if strings.TrimSpace(slots.AcceptanceCriteria) == "" {
		missing = append(missing, "AcceptanceCriteria")
	}
	if strings.TrimSpace(slots.CurrentPhase) == "" {
		missing = append(missing, "CurrentPhase")
	}
	if len(slots.ArtifactRefs) == 0 {
		missing = append(missing, "ArtifactRefs")
	}

	if len(missing) > 0 {
		return domain.NewEngineError(
			domain.ErrCompactionInvalid.Code,
			fmt.Sprintf("%s: missing slots: %s", domain.ErrCompactionInvalid.Message, strings.Join(missing, ", ")),
		)
	}
	return nil
}
