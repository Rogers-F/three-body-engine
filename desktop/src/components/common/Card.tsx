import { memo, type ReactNode } from 'react'

interface CardProps {
  children: ReactNode
  className?: string
  header?: ReactNode
  footer?: ReactNode
}

export const Card = memo(function Card({ children, className = '', header, footer }: CardProps) {
  return (
    <div
      className={[
        'bg-[var(--bg-card)] border border-[var(--border)] rounded-[12px] shadow-[0_1px_3px_rgba(0,0,0,0.08),0_1px_2px_rgba(0,0,0,0.04)]',
        className,
      ].join(' ')}
    >
      {header && (
        <div className="px-4 py-3 border-b border-[var(--border)]">
          {header}
        </div>
      )}
      <div className="p-4">{children}</div>
      {footer && (
        <div className="px-4 py-3 border-t border-[var(--border)]">
          {footer}
        </div>
      )}
    </div>
  )
})
