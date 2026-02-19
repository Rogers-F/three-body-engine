import { create } from 'zustand'
import type { Phase, FlowState, WorkerSpec, WorkflowEvent, ScoreCard, CostDelta } from '@/types/workflow'

interface WorkflowStore {
  flow: FlowState | null
  workers: WorkerSpec[]
  events: WorkflowEvent[]
  scoreCards: ScoreCard[]
  costDeltas: CostDelta[]

  setFlow: (flow: FlowState) => void
  addEvent: (event: WorkflowEvent) => void
  setWorkers: (workers: WorkerSpec[]) => void
  addScoreCard: (card: ScoreCard) => void
  addCostDelta: (delta: CostDelta) => void
  reset: () => void

  currentPhase: () => Phase | null
  totalCost: () => number
  isBlocked: () => boolean
}

const initialState = {
  flow: null,
  workers: [],
  events: [],
  scoreCards: [],
  costDeltas: [],
}

export const useWorkflowStore = create<WorkflowStore>()((set, get) => ({
  ...initialState,

  setFlow: (flow) => set({ flow }),

  addEvent: (event) => set((state) => ({
    events: [...state.events, event],
  })),

  setWorkers: (workers) => set({ workers }),

  addScoreCard: (card) => set((state) => ({
    scoreCards: [...state.scoreCards, card],
  })),

  addCostDelta: (delta) => set((state) => ({
    costDeltas: [...state.costDeltas, delta],
  })),

  reset: () => set(initialState),

  currentPhase: () => get().flow?.currentPhase ?? null,

  totalCost: () => get().costDeltas.reduce((sum, d) => sum + d.amountUsd, 0),

  isBlocked: () => get().flow?.status === 'blocked',
}))
