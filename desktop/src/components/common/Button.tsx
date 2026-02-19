import { memo, type ButtonHTMLAttributes, type ReactNode } from 'react'

type ButtonVariant = 'primary' | 'secondary' | 'ghost'
type ButtonSize = 'sm' | 'md' | 'lg'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: ButtonSize
  children: ReactNode
}

const variantClasses: Record<ButtonVariant, string> = {
  primary: [
    'bg-[var(--accent)] text-white',
    'hover:bg-[var(--accent-hover)]',
    'active:bg-[var(--accent-text)] active:scale-[0.98]',
    'focus-visible:ring-2 focus-visible:ring-[var(--accent)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--bg-primary)]',
    'disabled:opacity-40 disabled:pointer-events-none',
  ].join(' '),
  secondary: [
    'border border-[var(--accent)] text-[var(--accent-text)] bg-transparent',
    'hover:bg-[var(--accent)]/10',
    'active:bg-[var(--accent)]/20 active:scale-[0.98]',
    'focus-visible:ring-2 focus-visible:ring-[var(--accent)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--bg-primary)]',
    'disabled:opacity-40 disabled:pointer-events-none',
  ].join(' '),
  ghost: [
    'text-[var(--text-secondary)] bg-transparent',
    'hover:text-[var(--accent)] hover:bg-[var(--accent)]/5',
    'active:bg-[var(--accent)]/10 active:scale-[0.98]',
    'focus-visible:ring-2 focus-visible:ring-[var(--accent)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--bg-primary)]',
    'disabled:opacity-40 disabled:pointer-events-none',
  ].join(' '),
}

const sizeClasses: Record<ButtonSize, string> = {
  sm: 'px-3 py-1.5 text-xs rounded-[8px]',
  md: 'px-4 py-2 text-sm rounded-[8px]',
  lg: 'px-6 py-2.5 text-base rounded-[8px]',
}

export const Button = memo(function Button({
  variant = 'primary',
  size = 'md',
  className = '',
  children,
  ...props
}: ButtonProps) {
  return (
    <button
      className={[
        'inline-flex items-center justify-center font-medium transition-all duration-150 outline-none cursor-pointer',
        variantClasses[variant],
        sizeClasses[size],
        className,
      ].join(' ')}
      {...props}
    >
      {children}
    </button>
  )
})
