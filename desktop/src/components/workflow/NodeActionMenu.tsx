import { memo, useCallback, useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'

interface NodeActionMenuProps {
  x: number
  y: number
  nodeId: string
  onClose: () => void
  onAction: (action: string) => void
}

const menuActions = [
  { id: 'retry', label: 'Retry Phase' },
  { id: 'force-pass', label: 'Force Pass' },
  { id: 'rollback', label: 'Rollback' },
  { id: 'details', label: 'View Details' },
]

export const NodeActionMenu = memo(function NodeActionMenu({
  x,
  y,
  onClose,
  onAction,
}: NodeActionMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null)

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    },
    [onClose],
  )

  const handleClickOutside = useCallback(
    (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose()
      }
    },
    [onClose],
  )

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown)
    document.addEventListener('mousedown', handleClickOutside)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [handleKeyDown, handleClickOutside])

  return createPortal(
    <div
      ref={menuRef}
      className="fixed z-50 min-w-[160px] rounded-[8px] border border-[var(--border)] bg-[var(--bg-card)] py-1"
      style={{
        left: x,
        top: y,
        boxShadow: '0 4px 12px rgba(0,0,0,0.12), 0 1px 3px rgba(0,0,0,0.08)',
      }}
      role="menu"
    >
      {menuActions.map((action) => (
        <button
          key={action.id}
          onClick={() => {
            onAction(action.id)
            onClose()
          }}
          className="flex w-full items-center px-3 py-1.5 text-sm text-[var(--text-primary)] hover:bg-[var(--bg-secondary)] transition-colors cursor-pointer outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)] focus-visible:ring-inset"
          role="menuitem"
        >
          {action.label}
        </button>
      ))}
    </div>,
    document.body,
  )
})
