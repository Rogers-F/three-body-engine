import { memo } from 'react'
import { ProgressBar } from '@/components/common/ProgressBar'

interface BudgetGaugeProps {
  usedUsd: number
  capUsd: number
}

export const BudgetGauge = memo(function BudgetGauge({ usedUsd, capUsd }: BudgetGaugeProps) {
  const pct = capUsd > 0 ? (usedUsd / capUsd) * 100 : 0

  return (
    <div className="p-4 rounded-[12px] bg-[var(--bg-card)] border border-[var(--border)]">
      <ProgressBar
        value={pct}
        label={`$${usedUsd.toFixed(2)} / $${capUsd.toFixed(2)}`}
        showPercentage
      />
    </div>
  )
})
