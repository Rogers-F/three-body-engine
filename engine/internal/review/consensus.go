package review

import "github.com/anthropics/three-body-engine/internal/domain"

// ConsensusEngine aggregates multiple ScoreCards into a single ConsensusResult
// using weighted averaging.
type ConsensusEngine struct {
	Weights   map[string]float64
	Validator *SchemaValidator
}

// DefaultWeights returns the standard reviewer weight distribution.
func DefaultWeights() map[string]float64 {
	return map[string]float64{
		"primary":   0.45,
		"secondary": 0.25,
		"lead":      0.30,
	}
}

// NewConsensusEngine creates a ConsensusEngine with the given weight map.
func NewConsensusEngine(weights map[string]float64) *ConsensusEngine {
	return &ConsensusEngine{
		Weights:   weights,
		Validator: &SchemaValidator{},
	}
}

// Evaluate computes a weighted consensus from the provided score cards.
func (e *ConsensusEngine) Evaluate(cards []domain.ScoreCard) (*domain.ConsensusResult, error) {
	if len(cards) == 0 {
		return nil, domain.ErrConsensusNoCards
	}

	for _, card := range cards {
		if err := e.Validator.Validate(card); err != nil {
			return nil, err
		}
	}

	var weightedSum, totalWeight float64
	for _, card := range cards {
		avg := float64(card.Scores.Correctness+card.Scores.Security+
			card.Scores.Maintainability+card.Scores.Cost+
			card.Scores.DeliveryRisk) / 5.0

		weight := 1.0
		if w, ok := e.Weights[card.Reviewer]; ok {
			weight = w
		}
		weightedSum += avg * weight
		totalWeight += weight
	}

	finalScore := weightedSum / totalWeight

	var verdict string
	switch {
	case finalScore >= 4.0:
		verdict = "pass"
	case finalScore >= 3.0:
		verdict = "conditional_pass"
	default:
		verdict = "fail"
	}

	return &domain.ConsensusResult{
		WeightedScore: finalScore,
		FinalVerdict:  verdict,
		Blocking:      false,
		BlockReasons:  nil,
	}, nil
}
