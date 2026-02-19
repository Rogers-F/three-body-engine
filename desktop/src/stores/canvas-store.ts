import { create } from 'zustand'

interface Viewport {
  x: number
  y: number
  zoom: number
}

interface CanvasStore {
  viewport: Viewport
  selectedNodeId: string | null
  setViewport: (v: Viewport) => void
  selectNode: (id: string | null) => void
}

export const useCanvasStore = create<CanvasStore>()((set) => ({
  viewport: { x: 0, y: 0, zoom: 1 },
  selectedNodeId: null,
  setViewport: (viewport) => set({ viewport }),
  selectNode: (selectedNodeId) => set({ selectedNodeId }),
}))
