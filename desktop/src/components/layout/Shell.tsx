import { memo } from 'react'
import { Sidebar } from './Sidebar'
import { TopBar } from './TopBar'
import { DashboardView } from '@/views/DashboardView'
import { ReviewView } from '@/views/ReviewView'
import { CostView } from '@/views/CostView'
import { SettingsView } from '@/views/SettingsView'
import { useUIStore } from '@/stores/ui-store'

export const Shell = memo(function Shell() {
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
          {activeNav === 'dashboard' && <DashboardView />}
          {activeNav === 'review' && <ReviewView />}
          {activeNav === 'cost' && <CostView />}
          {activeNav === 'settings' && <SettingsView />}
        </main>
      </div>
    </div>
  )
})
