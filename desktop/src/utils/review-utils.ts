import type { Scores, ScoreCard } from '@/types/workflow'

export function averageScore(scores: Scores): number {
  return (
    (scores.correctness +
      scores.security +
      scores.maintainability +
      scores.cost +
      scores.deliveryRisk) /
    5
  )
}

const DEFAULT_WEIGHTS: Record<string, number> = {
  primary: 0.45,
  secondary: 0.25,
  lead: 0.3,
}

export function computeWeightedScore(
  cards: ScoreCard[],
  weights: Record<string, number> = DEFAULT_WEIGHTS,
): number {
  if (cards.length === 0) return 0

  let weightedSum = 0
  let totalWeight = 0

  for (const card of cards) {
    const w = weights[card.reviewer] ?? 1.0
    weightedSum += averageScore(card.scores) * w
    totalWeight += w
  }

  return totalWeight > 0 ? weightedSum / totalWeight : 0
}

export function deriveVerdict(score: number): 'pass' | 'conditional_pass' | 'fail' {
  if (score >= 4.0) return 'pass'
  if (score >= 3.0) return 'conditional_pass'
  return 'fail'
}
