package review

import (
	"math"
	"testing"

	"github.com/anthropics/three-body-engine/internal/domain"
)

func makeCard(reviewer string, c, s, m, cost, dr int, verdict string) domain.ScoreCard {
	return domain.ScoreCard{
		ReviewID: "rev-test",
		Reviewer: reviewer,
		Scores: domain.Scores{
			Correctness:     c,
			Security:        s,
			Maintainability: m,
			Cost:            cost,
			DeliveryRisk:    dr,
		},
		Verdict: verdict,
	}
}

func almostEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestEvaluate_SingleCard_Pass(t *testing.T) {
	eng := NewConsensusEngine(DefaultWeights())
	card := makeCard("primary", 5, 5, 4, 4, 4, "pass")
	// avg = (5+5+4+4+4)/5 = 4.4
	res, err := eng.Evaluate([]domain.ScoreCard{card})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.FinalVerdict != "pass" {
		t.Errorf("expected pass, got %s", res.FinalVerdict)
	}
	if !almostEqual(res.WeightedScore, 4.4, 0.01) {
		t.Errorf("expected score ~4.4, got %f", res.WeightedScore)
	}
}

func TestEvaluate_SingleCard_Fail(t *testing.T) {
	eng := NewConsensusEngine(DefaultWeights())
	card := makeCard("primary", 1, 1, 2, 2, 1, "fail")
	// avg = (1+1+2+2+1)/5 = 1.4
	res, err := eng.Evaluate([]domain.ScoreCard{card})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.FinalVerdict != "fail" {
		t.Errorf("expected fail, got %s", res.FinalVerdict)
	}
}

func TestEvaluate_SingleCard_ConditionalPass(t *testing.T) {
	eng := NewConsensusEngine(DefaultWeights())
	card := makeCard("primary", 3, 3, 3, 3, 3, "conditional_pass")
	// avg = 3.0
	res, err := eng.Evaluate([]domain.ScoreCard{card})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.FinalVerdict != "conditional_pass" {
		t.Errorf("expected conditional_pass, got %s", res.FinalVerdict)
	}
	if !almostEqual(res.WeightedScore, 3.0, 0.01) {
		t.Errorf("expected score ~3.0, got %f", res.WeightedScore)
	}
}

func TestEvaluate_MultipleCards_Weighted(t *testing.T) {
	eng := NewConsensusEngine(DefaultWeights())
	cards := []domain.ScoreCard{
		makeCard("primary", 5, 5, 5, 5, 5, "pass"),   // avg=5.0, weight=0.45
		makeCard("secondary", 3, 3, 3, 3, 3, "conditional_pass"), // avg=3.0, weight=0.25
		makeCard("lead", 4, 4, 4, 4, 4, "pass"),       // avg=4.0, weight=0.30
	}
	// weighted = (5.0*0.45 + 3.0*0.25 + 4.0*0.30) / (0.45+0.25+0.30) = (2.25+0.75+1.20)/1.0 = 4.2
	res, err := eng.Evaluate(cards)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !almostEqual(res.WeightedScore, 4.2, 0.01) {
		t.Errorf("expected score ~4.2, got %f", res.WeightedScore)
	}
	if res.FinalVerdict != "pass" {
		t.Errorf("expected pass, got %s", res.FinalVerdict)
	}
}

func TestEvaluate_UnknownReviewer_FallbackWeight(t *testing.T) {
	eng := NewConsensusEngine(DefaultWeights())
	card := makeCard("unknown-reviewer", 4, 4, 4, 4, 4, "pass")
	// avg = 4.0, fallback weight = 1.0
	res, err := eng.Evaluate([]domain.ScoreCard{card})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !almostEqual(res.WeightedScore, 4.0, 0.01) {
		t.Errorf("expected score ~4.0, got %f", res.WeightedScore)
	}
	if res.FinalVerdict != "pass" {
		t.Errorf("expected pass, got %s", res.FinalVerdict)
	}
}

func TestEvaluate_EmptyCards(t *testing.T) {
	eng := NewConsensusEngine(DefaultWeights())
	_, err := eng.Evaluate(nil)
	if err == nil {
		t.Fatal("expected error for empty cards")
	}
	engErr, ok := err.(*domain.EngineError)
	if !ok {
		t.Fatalf("expected *domain.EngineError, got %T", err)
	}
	if engErr.Code != domain.ErrConsensusNoCards.Code {
		t.Errorf("expected code %d, got %d", domain.ErrConsensusNoCards.Code, engErr.Code)
	}
}

func TestEvaluate_InvalidCard(t *testing.T) {
	eng := NewConsensusEngine(DefaultWeights())
	card := domain.ScoreCard{} // empty = invalid
	_, err := eng.Evaluate([]domain.ScoreCard{card})
	if err == nil {
		t.Fatal("expected validation error for invalid card")
	}
	engErr, ok := err.(*domain.EngineError)
	if !ok {
		t.Fatalf("expected *domain.EngineError, got %T", err)
	}
	if engErr.Code != domain.ErrScoreCardInvalid.Code {
		t.Errorf("expected code %d, got %d", domain.ErrScoreCardInvalid.Code, engErr.Code)
	}
}

func TestEvaluate_AllPerfectScores(t *testing.T) {
	eng := NewConsensusEngine(DefaultWeights())
	cards := []domain.ScoreCard{
		makeCard("primary", 5, 5, 5, 5, 5, "pass"),
		makeCard("secondary", 5, 5, 5, 5, 5, "pass"),
		makeCard("lead", 5, 5, 5, 5, 5, "pass"),
	}
	res, err := eng.Evaluate(cards)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !almostEqual(res.WeightedScore, 5.0, 0.01) {
		t.Errorf("expected score 5.0, got %f", res.WeightedScore)
	}
	if res.FinalVerdict != "pass" {
		t.Errorf("expected pass, got %s", res.FinalVerdict)
	}
}

func TestEvaluate_AllMinimumScores(t *testing.T) {
	eng := NewConsensusEngine(DefaultWeights())
	cards := []domain.ScoreCard{
		makeCard("primary", 1, 1, 1, 1, 1, "fail"),
		makeCard("secondary", 1, 1, 1, 1, 1, "fail"),
		makeCard("lead", 1, 1, 1, 1, 1, "fail"),
	}
	res, err := eng.Evaluate(cards)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !almostEqual(res.WeightedScore, 1.0, 0.01) {
		t.Errorf("expected score 1.0, got %f", res.WeightedScore)
	}
	if res.FinalVerdict != "fail" {
		t.Errorf("expected fail, got %s", res.FinalVerdict)
	}
}

func TestEvaluate_DefaultWeights(t *testing.T) {
	w := DefaultWeights()
	expected := map[string]float64{
		"primary":   0.45,
		"secondary": 0.25,
		"lead":      0.30,
	}
	for k, v := range expected {
		got, ok := w[k]
		if !ok {
			t.Errorf("missing weight for %s", k)
			continue
		}
		if !almostEqual(got, v, 0.001) {
			t.Errorf("weight for %s: expected %f, got %f", k, v, got)
		}
	}
	if len(w) != len(expected) {
		t.Errorf("expected %d weights, got %d", len(expected), len(w))
	}
}
