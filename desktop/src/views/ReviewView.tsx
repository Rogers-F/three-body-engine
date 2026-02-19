import { memo } from 'react'
import { mockScoreCards } from '@/data/mock-workflow'
import { useWorkflowStore } from '@/stores/workflow-store'
import { ConsensusBar } from '@/components/review/ConsensusBar'
import { ScorecardGrid } from '@/components/review/ScorecardGrid'
import { IssueList } from '@/components/review/IssueList'
import { useReviews } from '@/api/hooks'

export const ReviewView = memo(function ReviewView() {
  const taskId = useWorkflowStore((s) => s.taskId)
  const { reviews, error: reviewsError } = useReviews(taskId)

  const cards = taskId && !reviewsError && reviews.length > 0 ? reviews : mockScoreCards

  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold text-[var(--text-primary)]">Review</h2>
      <ConsensusBar cards={cards} />
      <ScorecardGrid cards={cards} />
      <div>
        <h3 className="text-sm font-medium text-[var(--text-primary)] mb-3">Issues</h3>
        <IssueList cards={cards} />
      </div>
    </div>
  )
})
