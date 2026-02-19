import { memo } from 'react'

interface ProgressBarProps {
  value: number
  label?: string
  showPercentage?: boolean
  className?: string
}

function getBarColor(value: number): string {
  if (value >= 100) return 'bg-[var(--phase-error)]'
  if (value >= 80) return 'bg-[var(--phase-warning)]'
  return 'bg-[var(--accent)]'
}

export const ProgressBar = memo(function ProgressBar({
  value,
  label,
  showPercentage = false,
  className = '',
}: ProgressBarProps) {
  const clamped = Math.max(0, Math.min(100, value))

  return (
    <div className={['w-full', className].join(' ')}>
      {(label || showPercentage) && (
        <div className="flex items-center justify-between mb-1.5 text-xs text-[var(--text-secondary)]">
          {label && <span>{label}</span>}
          {showPercentage && <span>{Math.round(clamped)}%</span>}
        </div>
      )}
      <div className="h-2 w-full rounded-full bg-[var(--border)] overflow-hidden">
        <div
          className={[
            'h-full rounded-full transition-all duration-300 ease-out',
            getBarColor(clamped),
          ].join(' ')}
          style={{ width: `${clamped}%` }}
          role="progressbar"
          aria-valuenow={clamped}
          aria-valuemin={0}
          aria-valuemax={100}
        />
      </div>
    </div>
  )
})
