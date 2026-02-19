import { memo } from 'react'
import { mockFlowState, mockCostDeltas } from '@/data/mock-workflow'
import { useWorkflowStore } from '@/stores/workflow-store'
import { Card } from '@/components/common/Card'
import { BudgetGauge } from '@/components/cost/BudgetGauge'
import { PhaseCostBreakdown } from '@/components/cost/PhaseCostBreakdown'
import { TokenChart } from '@/components/cost/TokenChart'
import { useCost } from '@/api/hooks'

export const CostView = memo(function CostView() {
  const taskId = useWorkflowStore((s) => s.taskId)
  const { cost, error: costError } = useCost(taskId)

  const usedUsd = cost && !costError ? cost.budgetUsedUsd : mockFlowState.budgetUsedUsd
  const capUsd = cost && !costError ? cost.budgetCapUsd : mockFlowState.budgetCapUsd
  const deltas = cost && !costError ? cost.deltas : mockCostDeltas

  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold text-[var(--text-primary)]">Cost</h2>
      <BudgetGauge usedUsd={usedUsd} capUsd={capUsd} />
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Card header={<span className="text-sm font-medium text-[var(--text-primary)]">Cost by Phase</span>}>
          <PhaseCostBreakdown costDeltas={deltas} />
        </Card>
        <Card header={<span className="text-sm font-medium text-[var(--text-primary)]">Token Usage (Cumulative)</span>}>
          <TokenChart costDeltas={deltas} />
        </Card>
      </div>
    </div>
  )
})
