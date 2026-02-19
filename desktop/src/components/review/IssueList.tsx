import { memo, useMemo } from 'react'
import type { ScoreCard, Issue } from '@/types/workflow'
import { Badge } from '@/components/common/Badge'

interface IssueListProps {
  cards: ScoreCard[]
}

const severityOrder: Record<string, number> = { P0: 0, P1: 1, P2: 2 }

const severityVariant = {
  P0: 'error',
  P1: 'warning',
  P2: 'pending',
} as const

export const IssueList = memo(function IssueList({ cards }: IssueListProps) {
  const grouped = useMemo(() => {
    const allIssues = cards.flatMap((c) => c.issues)
    allIssues.sort((a, b) => (severityOrder[a.severity] ?? 99) - (severityOrder[b.severity] ?? 99))

    const groups = new Map<string, Issue[]>()
    for (const issue of allIssues) {
      const list = groups.get(issue.severity) ?? []
      list.push(issue)
      groups.set(issue.severity, list)
    }

    return Array.from(groups.entries()).sort(
      ([a], [b]) => (severityOrder[a] ?? 99) - (severityOrder[b] ?? 99),
    )
  }, [cards])

  if (grouped.length === 0) {
    return (
      <p className="text-sm text-[var(--text-muted)]">No issues found.</p>
    )
  }

  return (
    <div className="space-y-4">
      {grouped.map(([severity, issues]) => (
        <div key={severity}>
          <div className="flex items-center gap-2 mb-2">
            <Badge variant={severityVariant[severity as keyof typeof severityVariant] ?? 'pending'}>
              {severity}
            </Badge>
            <span className="text-xs text-[var(--text-muted)]">
              {issues.length} issue{issues.length !== 1 ? 's' : ''}
            </span>
          </div>
          <div className="space-y-2">
            {issues.map((issue, idx) => (
              <div
                key={`${severity}-${idx}`}
                className="p-3 rounded-lg bg-[var(--bg-elevated)] border border-[var(--border)]"
              >
                <p className="text-xs font-mono text-[var(--text-muted)] mb-1">
                  {issue.location}
                </p>
                <p className="text-sm text-[var(--text-primary)] mb-1">
                  {issue.description}
                </p>
                <p className="text-xs text-[var(--text-secondary)]">
                  Suggestion: {issue.suggestion}
                </p>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
})
