import { memo } from 'react'
import { ThemeToggle } from '../common/ThemeToggle'

interface TopBarProps {
  className?: string
}

export const TopBar = memo(function TopBar({ className = '' }: TopBarProps) {
  return (
    <header
      className={[
        'flex items-center justify-between h-12 px-4 border-b border-[var(--border)] bg-[var(--bg-primary)]',
        className,
      ].join(' ')}
    >
      <div className="flex items-center gap-2">
        <h1 className="text-sm font-semibold text-[var(--text-primary)]">
          Three-Body Engine
        </h1>
      </div>

      <div className="flex items-center gap-2">
        <ThemeToggle />
      </div>
    </header>
  )
})
