import { memo, useState } from 'react'
import { LayoutDashboard, FileSearch, DollarSign, Settings, ChevronLeft, ChevronRight } from 'lucide-react'

interface NavItem {
  id: string
  label: string
  icon: typeof LayoutDashboard
}

const navItems: NavItem[] = [
  { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { id: 'review', label: 'Review', icon: FileSearch },
  { id: 'cost', label: 'Cost Monitor', icon: DollarSign },
  { id: 'settings', label: 'Settings', icon: Settings },
]

interface SidebarProps {
  activeItem?: string
  onNavigate?: (id: string) => void
}

export const Sidebar = memo(function Sidebar({ activeItem = 'dashboard', onNavigate }: SidebarProps) {
  const [collapsed, setCollapsed] = useState(false)

  return (
    <aside
      className={[
        'flex flex-col h-full border-r border-[var(--border)] bg-[var(--bg-secondary)] transition-[width] duration-200',
        collapsed ? 'w-16' : 'w-60',
      ].join(' ')}
    >
      <div className="flex items-center justify-end p-2">
        <button
          onClick={() => setCollapsed(!collapsed)}
          className={[
            'inline-flex items-center justify-center w-8 h-8 rounded-[8px] transition-all duration-150 outline-none cursor-pointer',
            'text-[var(--text-secondary)] bg-transparent',
            'hover:text-[var(--accent)] hover:bg-[var(--accent)]/5',
            'active:bg-[var(--accent)]/10 active:scale-[0.98]',
            'focus-visible:ring-2 focus-visible:ring-[var(--accent)] focus-visible:ring-offset-2',
          ].join(' ')}
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          {collapsed ? <ChevronRight size={16} /> : <ChevronLeft size={16} />}
        </button>
      </div>

      <nav className="flex flex-col gap-1 px-2 flex-1">
        {navItems.map((item) => {
          const isActive = item.id === activeItem
          const Icon = item.icon
          return (
            <button
              key={item.id}
              onClick={() => onNavigate?.(item.id)}
              className={[
                'flex items-center gap-3 px-3 py-2 rounded-[8px] text-sm font-medium transition-all duration-150 outline-none cursor-pointer',
                isActive
                  ? 'bg-[var(--accent)]/10 text-[var(--accent)] border-l-2 border-[var(--accent)]'
                  : 'text-[var(--text-secondary)] border-l-2 border-transparent',
                !isActive && 'hover:text-[var(--text-primary)] hover:bg-[var(--bg-card)]',
                'active:scale-[0.98]',
                'focus-visible:ring-2 focus-visible:ring-[var(--accent)] focus-visible:ring-offset-2',
              ].join(' ')}
              aria-current={isActive ? 'page' : undefined}
            >
              <Icon size={18} className="shrink-0" />
              {!collapsed && <span>{item.label}</span>}
            </button>
          )
        })}
      </nav>
    </aside>
  )
})
