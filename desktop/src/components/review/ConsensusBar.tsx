import { memo, useMemo } from 'react'
import type { ScoreCard } from '@/types/workflow'
import { Badge } from '@/components/common/Badge'
import { ProgressBar } from '@/components/common/ProgressBar'
import { computeWeightedScore, deriveVerdict } from '@/utils/review-utils'

interface ConsensusBarProps {
  cards: ScoreCard[]
}

const verdictVariant = {
  pass: 'completed',
  conditional_pass: 'warning',
  fail: 'error',
} as const

export const ConsensusBar = memo(function ConsensusBar({ cards }: ConsensusBarProps) {
  const { score, verdict } = useMemo(() => {
    const s = computeWeightedScore(cards)
    return { score: s, verdict: deriveVerdict(s) }
  }, [cards])

  return (
    <div className="flex items-center gap-4 p-4 rounded-[12px] bg-[var(--bg-card)] border border-[var(--border)]">
      <span className="text-2xl font-bold text-[var(--text-primary)] tabular-nums">
        {score.toFixed(2)}
      </span>
      <Badge variant={verdictVariant[verdict]}>
        {verdict.replace('_', ' ')}
      </Badge>
      <div className="flex-1">
        <ProgressBar value={score * 20} label="Weighted Consensus" showPercentage />
      </div>
    </div>
  )
})
