import { create } from 'zustand'

type DrawerContent = 'phase' | 'review' | 'cost'

interface UIStore {
  sidebarCollapsed: boolean
  drawerOpen: boolean
  drawerContent: DrawerContent | null
  zenMode: boolean
  activeNav: string
  toggleSidebar: () => void
  openDrawer: (content: DrawerContent) => void
  closeDrawer: () => void
  toggleZenMode: () => void
  setActiveNav: (id: string) => void
}

export const useUIStore = create<UIStore>()((set) => ({
  sidebarCollapsed: false,
  drawerOpen: false,
  drawerContent: null,
  zenMode: false,
  activeNav: 'dashboard',

  toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),

  openDrawer: (content) => set({ drawerOpen: true, drawerContent: content }),

  closeDrawer: () => set({ drawerOpen: false, drawerContent: null }),

  toggleZenMode: () => set((state) => ({ zenMode: !state.zenMode })),

  setActiveNav: (activeNav) => set({ activeNav }),
}))
