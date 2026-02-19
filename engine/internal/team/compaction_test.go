package team

import (
	"context"
	"strings"
	"testing"

	"github.com/anthropics/three-body-engine/internal/domain"
)

func validSlots() domain.CompactionSlots {
	return domain.CompactionSlots{
		TaskSpec:           "implement feature X",
		AcceptanceCriteria: "all tests pass",
		CurrentPhase:       "C",
		ArtifactRefs: []domain.ArtifactRef{
			{ID: "a1", Type: "file", Path: "main.go", Version: 1, Hash: "abc"},
		},
	}
}

func TestCompactionValidator_ValidSlots(t *testing.T) {
	v := &CompactionValidator{}
	if err := v.Validate(context.Background(), validSlots()); err != nil {
		t.Errorf("expected nil error for valid slots, got: %v", err)
	}
}

func TestCompactionValidator_EmptyTaskSpec(t *testing.T) {
	v := &CompactionValidator{}
	slots := validSlots()
	slots.TaskSpec = ""
	err := v.Validate(context.Background(), slots)
	if err == nil {
		t.Fatal("expected error for empty TaskSpec")
	}
	if !strings.Contains(err.Error(), "TaskSpec") {
		t.Errorf("error should mention TaskSpec, got: %v", err)
	}
}

func TestCompactionValidator_EmptyAcceptanceCriteria(t *testing.T) {
	v := &CompactionValidator{}
	slots := validSlots()
	slots.AcceptanceCriteria = ""
	err := v.Validate(context.Background(), slots)
	if err == nil {
		t.Fatal("expected error for empty AcceptanceCriteria")
	}
	if !strings.Contains(err.Error(), "AcceptanceCriteria") {
		t.Errorf("error should mention AcceptanceCriteria, got: %v", err)
	}
}

func TestCompactionValidator_EmptyCurrentPhase(t *testing.T) {
	v := &CompactionValidator{}
	slots := validSlots()
	slots.CurrentPhase = ""
	err := v.Validate(context.Background(), slots)
	if err == nil {
		t.Fatal("expected error for empty CurrentPhase")
	}
	if !strings.Contains(err.Error(), "CurrentPhase") {
		t.Errorf("error should mention CurrentPhase, got: %v", err)
	}
}

func TestCompactionValidator_EmptyArtifactRefs(t *testing.T) {
	v := &CompactionValidator{}
	slots := validSlots()
	slots.ArtifactRefs = nil
	err := v.Validate(context.Background(), slots)
	if err == nil {
		t.Fatal("expected error for empty ArtifactRefs")
	}
	if !strings.Contains(err.Error(), "ArtifactRefs") {
		t.Errorf("error should mention ArtifactRefs, got: %v", err)
	}
}

func TestCompactionValidator_MultipleMissing(t *testing.T) {
	v := &CompactionValidator{}
	slots := domain.CompactionSlots{} // all empty
	err := v.Validate(context.Background(), slots)
	if err == nil {
		t.Fatal("expected error for all-empty slots")
	}
	msg := err.Error()
	for _, field := range []string{"TaskSpec", "AcceptanceCriteria", "CurrentPhase", "ArtifactRefs"} {
		if !strings.Contains(msg, field) {
			t.Errorf("error should mention %s, got: %v", field, msg)
		}
	}
}
