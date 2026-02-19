package review

import (
	"fmt"

	"github.com/anthropics/three-body-engine/internal/domain"
)

// BlockerChecker inspects score cards for blocking conditions that must be
// resolved before a workflow can proceed.
type BlockerChecker struct{}

// Check examines all cards for critically low scores and P0 issues.
// It returns whether any blocking condition was found and the list of reasons.
func (c *BlockerChecker) Check(cards []domain.ScoreCard) (blocking bool, reasons []string) {
	for _, card := range cards {
		if card.Scores.Correctness <= 2 {
			reasons = append(reasons, fmt.Sprintf(
				"%s: correctness score %d is critically low",
				card.Reviewer, card.Scores.Correctness))
		}
		if card.Scores.Security <= 2 {
			reasons = append(reasons, fmt.Sprintf(
				"%s: security score %d is critically low",
				card.Reviewer, card.Scores.Security))
		}
		for _, issue := range card.Issues {
			if issue.Severity == "P0" {
				reasons = append(reasons, fmt.Sprintf(
					"%s: P0 issue at %s: %s",
					card.Reviewer, issue.Location, issue.Description))
			}
		}
	}
	return len(reasons) > 0, reasons
}
