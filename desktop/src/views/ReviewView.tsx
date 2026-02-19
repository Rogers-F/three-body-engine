import { memo } from 'react'
import { mockScoreCards } from '@/data/mock-workflow'
import { ConsensusBar } from '@/components/review/ConsensusBar'
import { ScorecardGrid } from '@/components/review/ScorecardGrid'
import { IssueList } from '@/components/review/IssueList'

export const ReviewView = memo(function ReviewView() {
  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold text-[var(--text-primary)]">Review</h2>
      <ConsensusBar cards={mockScoreCards} />
      <ScorecardGrid cards={mockScoreCards} />
      <div>
        <h3 className="text-sm font-medium text-[var(--text-primary)] mb-3">Issues</h3>
        <IssueList cards={mockScoreCards} />
      </div>
    </div>
  )
})
