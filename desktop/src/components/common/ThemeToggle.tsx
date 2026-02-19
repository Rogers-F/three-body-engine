import { memo } from 'react'
import { Sun, Moon } from 'lucide-react'
import { useTheme } from '../../design-system/ThemeProvider'

interface ThemeToggleProps {
  className?: string
}

export const ThemeToggle = memo(function ThemeToggle({ className = '' }: ThemeToggleProps) {
  const { resolved, toggle } = useTheme()

  return (
    <button
      onClick={toggle}
      className={[
        'inline-flex items-center justify-center w-8 h-8 rounded-[8px] transition-all duration-150 outline-none cursor-pointer',
        'text-[var(--text-secondary)] bg-transparent',
        'hover:text-[var(--accent)] hover:bg-[var(--accent)]/5',
        'active:bg-[var(--accent)]/10 active:scale-[0.98]',
        'focus-visible:ring-2 focus-visible:ring-[var(--accent)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--bg-primary)]',
        className,
      ].join(' ')}
      aria-label={`Switch to ${resolved === 'light' ? 'dark' : 'light'} mode`}
    >
      {resolved === 'light' ? <Moon size={16} /> : <Sun size={16} />}
    </button>
  )
})
