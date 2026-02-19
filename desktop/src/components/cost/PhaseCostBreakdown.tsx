import { memo, useMemo } from 'react'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import type { CostDelta } from '@/types/workflow'

interface PhaseCostBreakdownProps {
  costDeltas: CostDelta[]
}

export const PhaseCostBreakdown = memo(function PhaseCostBreakdown({ costDeltas }: PhaseCostBreakdownProps) {
  const data = useMemo(() => {
    const byPhase = new Map<string, number>()
    for (const d of costDeltas) {
      byPhase.set(d.phase, (byPhase.get(d.phase) ?? 0) + d.amountUsd)
    }
    return Array.from(byPhase.entries())
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([phase, amount]) => ({ phase, amount: Number(amount.toFixed(4)) }))
  }, [costDeltas])

  return (
    <ResponsiveContainer width="100%" height={300}>
      <BarChart data={data} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
        <XAxis dataKey="phase" tick={{ fill: 'var(--text-secondary)', fontSize: 12 }} />
        <YAxis tick={{ fill: 'var(--text-secondary)', fontSize: 12 }} />
        <Tooltip
          contentStyle={{
            backgroundColor: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            color: 'var(--text-primary)',
          }}
          formatter={(value: number) => [`$${value.toFixed(2)}`, 'Cost']}
        />
        <Bar dataKey="amount" fill="#6366f1" radius={[4, 4, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  )
})
