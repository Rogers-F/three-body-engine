import { memo } from 'react'
import type { ScoreCard } from '@/types/workflow'
import { Card } from '@/components/common/Card'
import { Badge } from '@/components/common/Badge'
import { ProgressBar } from '@/components/common/ProgressBar'

interface ScorecardCardProps {
  card: ScoreCard
}

const verdictVariant = {
  pass: 'completed',
  conditional_pass: 'warning',
  fail: 'error',
} as const

const scoreLabels: { key: keyof ScoreCard['scores']; label: string }[] = [
  { key: 'correctness', label: 'Correctness' },
  { key: 'security', label: 'Security' },
  { key: 'maintainability', label: 'Maintainability' },
  { key: 'cost', label: 'Cost' },
  { key: 'deliveryRisk', label: 'Delivery Risk' },
]

export const ScorecardCard = memo(function ScorecardCard({ card }: ScorecardCardProps) {
  return (
    <Card
      header={
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium text-[var(--text-primary)]">
            {card.reviewer}
          </span>
          <Badge variant={verdictVariant[card.verdict]}>
            {card.verdict.replace('_', ' ')}
          </Badge>
        </div>
      }
    >
      <div className="space-y-3">
        {scoreLabels.map(({ key, label }) => (
          <ProgressBar
            key={key}
            value={card.scores[key] * 20}
            label={label}
            showPercentage
          />
        ))}
        {card.issues.length > 0 && (
          <div className="pt-2 border-t border-[var(--border)]">
            <span className="text-xs text-[var(--text-muted)]">
              {card.issues.length} issue{card.issues.length !== 1 ? 's' : ''} found
            </span>
          </div>
        )}
      </div>
    </Card>
  )
})
