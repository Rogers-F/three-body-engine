import { memo } from 'react'
import type { ScoreCard } from '@/types/workflow'
import { ScorecardCard } from './ScorecardCard'

interface ScorecardGridProps {
  cards: ScoreCard[]
}

export const ScorecardGrid = memo(function ScorecardGrid({ cards }: ScorecardGridProps) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      {cards.map((card) => (
        <ScorecardCard key={card.reviewId} card={card} />
      ))}
    </div>
  )
})
