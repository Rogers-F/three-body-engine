package review

import (
	"strings"
	"testing"

	"github.com/anthropics/three-body-engine/internal/domain"
)

func validCard() domain.ScoreCard {
	return domain.ScoreCard{
		ReviewID: "rev-001",
		Reviewer: "primary",
		Scores: domain.Scores{
			Correctness:     4,
			Security:        5,
			Maintainability: 4,
			Cost:            3,
			DeliveryRisk:    4,
		},
		Issues: []domain.Issue{
			{Severity: "P1", Location: "main.go:10", Description: "minor issue"},
		},
		Alternatives: []string{"option-a"},
		Verdict:      "pass",
	}
}

func TestValidate_ValidCard(t *testing.T) {
	v := &SchemaValidator{}
	if err := v.Validate(validCard()); err != nil {
		t.Fatalf("expected nil error for valid card, got: %v", err)
	}
}

func TestValidate_EmptyReviewID(t *testing.T) {
	v := &SchemaValidator{}
	card := validCard()
	card.ReviewID = ""
	err := v.Validate(card)
	if err == nil {
		t.Fatal("expected error for empty ReviewID")
	}
	if !strings.Contains(err.Error(), "ReviewID must be non-empty") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidate_EmptyReviewer(t *testing.T) {
	v := &SchemaValidator{}
	card := validCard()
	card.Reviewer = ""
	err := v.Validate(card)
	if err == nil {
		t.Fatal("expected error for empty Reviewer")
	}
	if !strings.Contains(err.Error(), "Reviewer must be non-empty") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidate_InvalidVerdict(t *testing.T) {
	v := &SchemaValidator{}
	card := validCard()
	card.Verdict = "maybe"
	err := v.Validate(card)
	if err == nil {
		t.Fatal("expected error for invalid verdict")
	}
	if !strings.Contains(err.Error(), "Verdict") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidate_ScoreTooLow(t *testing.T) {
	v := &SchemaValidator{}
	card := validCard()
	card.Scores.Correctness = 0
	err := v.Validate(card)
	if err == nil {
		t.Fatal("expected error for score too low")
	}
	if !strings.Contains(err.Error(), "Correctness score 0 out of range") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidate_ScoreTooHigh(t *testing.T) {
	v := &SchemaValidator{}
	card := validCard()
	card.Scores.Security = 6
	err := v.Validate(card)
	if err == nil {
		t.Fatal("expected error for score too high")
	}
	if !strings.Contains(err.Error(), "Security score 6 out of range") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidate_InvalidIssueSeverity(t *testing.T) {
	v := &SchemaValidator{}
	card := validCard()
	card.Issues = []domain.Issue{
		{Severity: "P3", Location: "foo.go:1", Description: "bad severity"},
	}
	err := v.Validate(card)
	if err == nil {
		t.Fatal("expected error for invalid issue severity")
	}
	if !strings.Contains(err.Error(), "Issue[0] severity") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidate_MultipleViolations(t *testing.T) {
	v := &SchemaValidator{}
	card := domain.ScoreCard{
		ReviewID: "",
		Reviewer: "",
		Scores: domain.Scores{
			Correctness:     0,
			Security:        6,
			Maintainability: 3,
			Cost:            3,
			DeliveryRisk:    3,
		},
		Verdict: "invalid",
	}
	err := v.Validate(card)
	if err == nil {
		t.Fatal("expected error for multiple violations")
	}
	msg := err.Error()
	if !strings.Contains(msg, "ReviewID") {
		t.Error("missing ReviewID violation in error")
	}
	if !strings.Contains(msg, "Reviewer") {
		t.Error("missing Reviewer violation in error")
	}
	if !strings.Contains(msg, "Verdict") {
		t.Error("missing Verdict violation in error")
	}
	if !strings.Contains(msg, "Correctness") {
		t.Error("missing Correctness violation in error")
	}
	if !strings.Contains(msg, "Security") {
		t.Error("missing Security violation in error")
	}
}

func TestValidate_EmptyIssues(t *testing.T) {
	v := &SchemaValidator{}
	card := validCard()
	card.Issues = nil
	if err := v.Validate(card); err != nil {
		t.Fatalf("expected nil error when issues are empty, got: %v", err)
	}
}
