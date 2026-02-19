import { memo, type ReactNode } from 'react'

type BadgeVariant = 'pending' | 'active' | 'completed' | 'error' | 'warning'

interface BadgeProps {
  variant: BadgeVariant
  children: ReactNode
  className?: string
}

const variantClasses: Record<BadgeVariant, string> = {
  pending: 'bg-[var(--phase-pending)]/15 text-[var(--phase-pending)]',
  active: 'bg-[var(--phase-active)]/15 text-[var(--phase-active)]',
  completed: 'bg-[var(--phase-completed)]/15 text-[var(--phase-completed)]',
  error: 'bg-[var(--phase-error)]/15 text-[var(--phase-error)]',
  warning: 'bg-[var(--phase-warning)]/15 text-[var(--phase-warning)]',
}

export const Badge = memo(function Badge({ variant, children, className = '' }: BadgeProps) {
  return (
    <span
      className={[
        'inline-flex items-center px-2 py-0.5 text-xs font-medium rounded-full',
        variantClasses[variant],
        className,
      ].join(' ')}
    >
      {children}
    </span>
  )
})
