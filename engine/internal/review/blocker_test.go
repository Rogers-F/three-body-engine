package review

import (
	"strings"
	"testing"

	"github.com/anthropics/three-body-engine/internal/domain"
)

func safeCard(reviewer string) domain.ScoreCard {
	return domain.ScoreCard{
		ReviewID: "rev-safe",
		Reviewer: reviewer,
		Scores: domain.Scores{
			Correctness:     4,
			Security:        4,
			Maintainability: 4,
			Cost:            4,
			DeliveryRisk:    4,
		},
		Verdict: "pass",
	}
}

func TestCheck_NoBlockers(t *testing.T) {
	c := &BlockerChecker{}
	blocking, reasons := c.Check([]domain.ScoreCard{safeCard("primary")})
	if blocking {
		t.Fatalf("expected no blocking, got reasons: %v", reasons)
	}
	if len(reasons) != 0 {
		t.Fatalf("expected 0 reasons, got %d", len(reasons))
	}
}

func TestCheck_LowCorrectness(t *testing.T) {
	c := &BlockerChecker{}
	card := safeCard("primary")
	card.Scores.Correctness = 2
	blocking, reasons := c.Check([]domain.ScoreCard{card})
	if !blocking {
		t.Fatal("expected blocking for low correctness")
	}
	if len(reasons) != 1 {
		t.Fatalf("expected 1 reason, got %d: %v", len(reasons), reasons)
	}
	if !strings.Contains(reasons[0], "correctness score 2") {
		t.Errorf("unexpected reason: %s", reasons[0])
	}
}

func TestCheck_LowSecurity(t *testing.T) {
	c := &BlockerChecker{}
	card := safeCard("primary")
	card.Scores.Security = 1
	blocking, reasons := c.Check([]domain.ScoreCard{card})
	if !blocking {
		t.Fatal("expected blocking for low security")
	}
	if len(reasons) != 1 {
		t.Fatalf("expected 1 reason, got %d: %v", len(reasons), reasons)
	}
	if !strings.Contains(reasons[0], "security score 1") {
		t.Errorf("unexpected reason: %s", reasons[0])
	}
}

func TestCheck_P0Issue(t *testing.T) {
	c := &BlockerChecker{}
	card := safeCard("lead")
	card.Issues = []domain.Issue{
		{Severity: "P0", Location: "auth.go:42", Description: "credentials leaked"},
	}
	blocking, reasons := c.Check([]domain.ScoreCard{card})
	if !blocking {
		t.Fatal("expected blocking for P0 issue")
	}
	if len(reasons) != 1 {
		t.Fatalf("expected 1 reason, got %d: %v", len(reasons), reasons)
	}
	if !strings.Contains(reasons[0], "P0 issue at auth.go:42") {
		t.Errorf("unexpected reason: %s", reasons[0])
	}
	if !strings.Contains(reasons[0], "credentials leaked") {
		t.Errorf("expected description in reason: %s", reasons[0])
	}
}

func TestCheck_MultipleBlockers(t *testing.T) {
	c := &BlockerChecker{}
	card := safeCard("primary")
	card.Scores.Correctness = 1
	card.Scores.Security = 2
	card.Issues = []domain.Issue{
		{Severity: "P0", Location: "main.go:1", Description: "crash on start"},
	}
	blocking, reasons := c.Check([]domain.ScoreCard{card})
	if !blocking {
		t.Fatal("expected blocking for multiple issues")
	}
	if len(reasons) != 3 {
		t.Fatalf("expected 3 reasons, got %d: %v", len(reasons), reasons)
	}
}

func TestCheck_BothLowScores(t *testing.T) {
	c := &BlockerChecker{}
	card := safeCard("secondary")
	card.Scores.Correctness = 2
	card.Scores.Security = 2
	blocking, reasons := c.Check([]domain.ScoreCard{card})
	if !blocking {
		t.Fatal("expected blocking for both low scores")
	}
	if len(reasons) != 2 {
		t.Fatalf("expected 2 reasons, got %d: %v", len(reasons), reasons)
	}
}

func TestCheck_P0AndLowScore(t *testing.T) {
	c := &BlockerChecker{}
	card := safeCard("lead")
	card.Scores.Correctness = 1
	card.Issues = []domain.Issue{
		{Severity: "P0", Location: "db.go:99", Description: "data corruption"},
	}
	blocking, reasons := c.Check([]domain.ScoreCard{card})
	if !blocking {
		t.Fatal("expected blocking")
	}
	if len(reasons) != 2 {
		t.Fatalf("expected 2 reasons, got %d: %v", len(reasons), reasons)
	}
}

func TestCheck_HighScoresNoP0(t *testing.T) {
	c := &BlockerChecker{}
	card := safeCard("primary")
	card.Scores.Correctness = 5
	card.Scores.Security = 5
	card.Issues = []domain.Issue{
		{Severity: "P1", Location: "style.go:10", Description: "naming convention"},
		{Severity: "P2", Location: "docs.go:5", Description: "typo in comment"},
	}
	blocking, reasons := c.Check([]domain.ScoreCard{card})
	if blocking {
		t.Fatalf("expected no blocking, got reasons: %v", reasons)
	}
}

func TestCheck_MultipleCardsOneBlocking(t *testing.T) {
	c := &BlockerChecker{}
	good := safeCard("primary")
	bad := safeCard("secondary")
	bad.Scores.Security = 1
	blocking, reasons := c.Check([]domain.ScoreCard{good, bad})
	if !blocking {
		t.Fatal("expected blocking when one card has low security")
	}
	if len(reasons) != 1 {
		t.Fatalf("expected 1 reason, got %d: %v", len(reasons), reasons)
	}
	if !strings.Contains(reasons[0], "secondary") {
		t.Errorf("expected secondary in reason, got: %s", reasons[0])
	}
}
