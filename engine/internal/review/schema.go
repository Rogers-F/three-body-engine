package review

import (
	"fmt"
	"strings"

	"github.com/anthropics/three-body-engine/internal/domain"
)

// SchemaValidator validates ScoreCard fields against the review schema.
type SchemaValidator struct{}

var validVerdicts = map[string]bool{
	"pass":             true,
	"conditional_pass": true,
	"fail":             true,
}

var validSeverities = map[string]bool{
	"P0": true,
	"P1": true,
	"P2": true,
}

// Validate checks all fields of the given ScoreCard and returns an error
// listing all violations if any are found.
func (v *SchemaValidator) Validate(card domain.ScoreCard) error {
	var violations []string

	if card.ReviewID == "" {
		violations = append(violations, "ReviewID must be non-empty")
	}
	if card.Reviewer == "" {
		violations = append(violations, "Reviewer must be non-empty")
	}
	if !validVerdicts[card.Verdict] {
		violations = append(violations, fmt.Sprintf("Verdict %q is not valid; must be pass, conditional_pass, or fail", card.Verdict))
	}

	type scoreEntry struct {
		name  string
		value int
	}
	scores := []scoreEntry{
		{"Correctness", card.Scores.Correctness},
		{"Security", card.Scores.Security},
		{"Maintainability", card.Scores.Maintainability},
		{"Cost", card.Scores.Cost},
		{"DeliveryRisk", card.Scores.DeliveryRisk},
	}
	for _, s := range scores {
		if s.value < 1 || s.value > 5 {
			violations = append(violations, fmt.Sprintf("%s score %d out of range [1, 5]", s.name, s.value))
		}
	}

	for i, issue := range card.Issues {
		if !validSeverities[issue.Severity] {
			violations = append(violations, fmt.Sprintf("Issue[%d] severity %q is not valid; must be P0, P1, or P2", i, issue.Severity))
		}
	}

	if len(violations) > 0 {
		msg := strings.Join(violations, "; ")
		return domain.NewEngineError(domain.ErrScoreCardInvalid.Code, msg)
	}
	return nil
}
