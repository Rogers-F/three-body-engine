import { memo, useMemo } from 'react'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts'
import type { CostDelta } from '@/types/workflow'

interface TokenChartProps {
  costDeltas: CostDelta[]
}

export const TokenChart = memo(function TokenChart({ costDeltas }: TokenChartProps) {
  const data = useMemo(() => {
    const byPhase = new Map<string, { input: number; output: number }>()
    for (const d of costDeltas) {
      const cur = byPhase.get(d.phase) ?? { input: 0, output: 0 }
      cur.input += d.inputTokens
      cur.output += d.outputTokens
      byPhase.set(d.phase, cur)
    }

    const sorted = Array.from(byPhase.entries()).sort(([a], [b]) => a.localeCompare(b))

    let cumInput = 0
    let cumOutput = 0

    return sorted.map(([phase, tokens]) => {
      cumInput += tokens.input
      cumOutput += tokens.output
      return { phase, inputTokens: cumInput, outputTokens: cumOutput }
    })
  }, [costDeltas])

  return (
    <ResponsiveContainer width="100%" height={300}>
      <LineChart data={data} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
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
        />
        <Legend />
        <Line
          type="monotone"
          dataKey="inputTokens"
          name="Input Tokens"
          stroke="#6366f1"
          strokeWidth={2}
          dot={{ r: 4 }}
        />
        <Line
          type="monotone"
          dataKey="outputTokens"
          name="Output Tokens"
          stroke="#22c55e"
          strokeWidth={2}
          dot={{ r: 4 }}
        />
      </LineChart>
    </ResponsiveContainer>
  )
})
