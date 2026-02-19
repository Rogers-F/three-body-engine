import { memo, useState, type ReactNode } from 'react'
import { Sidebar } from './Sidebar'
import { TopBar } from './TopBar'

interface ShellProps {
  children?: ReactNode
}

export const Shell = memo(function Shell({ children }: ShellProps) {
  const [activeNav, setActiveNav] = useState('dashboard')

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-[var(--bg-primary)]">
      <Sidebar activeItem={activeNav} onNavigate={setActiveNav} />
      <div className="flex flex-col flex-1 min-w-0">
        <TopBar />
        <main className="flex-1 overflow-auto p-6">
          {children ?? (
            <div className="text-[var(--text-muted)] text-sm">
              Select a view from the sidebar.
            </div>
          )}
        </main>
      </div>
    </div>
  )
})
