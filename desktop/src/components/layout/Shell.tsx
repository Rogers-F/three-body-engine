import { memo, type ReactNode } from 'react'
import { Sidebar } from './Sidebar'
import { TopBar } from './TopBar'
import { WorkflowCanvas } from '@/components/workflow/WorkflowCanvas'
import { useUIStore } from '@/stores/ui-store'

interface ShellProps {
  children?: ReactNode
}

export const Shell = memo(function Shell({ children }: ShellProps) {
  const activeNav = useUIStore((s) => s.activeNav)
  const setActiveNav = useUIStore((s) => s.setActiveNav)
  const sidebarCollapsed = useUIStore((s) => s.sidebarCollapsed)
  const toggleSidebar = useUIStore((s) => s.toggleSidebar)

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-[var(--bg-primary)]">
      <Sidebar
        activeItem={activeNav}
        onNavigate={setActiveNav}
        collapsed={sidebarCollapsed}
        onToggle={toggleSidebar}
      />
      <div className="flex flex-col flex-1 min-w-0">
        <TopBar />
        <main className="flex-1 overflow-auto p-6">
          {activeNav === 'dashboard' ? (
            <WorkflowCanvas />
          ) : (
            children ?? (
              <div className="text-[var(--text-muted)] text-sm">
                Select a view from the sidebar.
              </div>
            )
          )}
        </main>
      </div>
    </div>
  )
})
