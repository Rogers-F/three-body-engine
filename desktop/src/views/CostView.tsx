import { memo } from 'react'
import { mockFlowState, mockCostDeltas } from '@/data/mock-workflow'
import { Card } from '@/components/common/Card'
import { BudgetGauge } from '@/components/cost/BudgetGauge'
import { PhaseCostBreakdown } from '@/components/cost/PhaseCostBreakdown'
import { TokenChart } from '@/components/cost/TokenChart'

export const CostView = memo(function CostView() {
  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold text-[var(--text-primary)]">Cost</h2>
      <BudgetGauge usedUsd={mockFlowState.budgetUsedUsd} capUsd={mockFlowState.budgetCapUsd} />
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Card header={<span className="text-sm font-medium text-[var(--text-primary)]">Cost by Phase</span>}>
          <PhaseCostBreakdown costDeltas={mockCostDeltas} />
        </Card>
        <Card header={<span className="text-sm font-medium text-[var(--text-primary)]">Token Usage (Cumulative)</span>}>
          <TokenChart costDeltas={mockCostDeltas} />
        </Card>
      </div>
    </div>
  )
})
