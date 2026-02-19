import type { Phase, WorkerSpec } from '@/types/workflow'
import { PHASES } from '@/types/workflow'

export interface LayoutNode {
  id: string
  type: 'phase' | 'worker'
  position: { x: number; y: number }
  data: Record<string, unknown>
  parentId?: string
}

export interface LayoutEdge {
  id: string
  source: string
  target: string
  type?: string
  animated?: boolean
  style?: Record<string, unknown>
  data?: Record<string, unknown>
}

type PhaseStatus = 'pending' | 'active' | 'completed' | 'error'

const PHASE_X_START = 100
const PHASE_X_GAP = 220
const PHASE_Y = 100
const WORKER_Y_OFFSET = 180
const WORKER_X_GAP = 160

const PHASE_ORDER: Phase[] = PHASES.map((p) => p.id)

function getPhaseIndex(phase: Phase): number {
  return PHASE_ORDER.indexOf(phase)
}

function derivePhaseStatus(phase: Phase, currentPhase: Phase, statusOverrides: Map<Phase, string>): PhaseStatus {
  const override = statusOverrides.get(phase)
  if (override === 'error') return 'error'

  const currentIdx = getPhaseIndex(currentPhase)
  const phaseIdx = getPhaseIndex(phase)

  if (phaseIdx < currentIdx) return 'completed'
  if (phaseIdx === currentIdx) return 'active'
  return 'pending'
}

export function buildPhaseNodes(currentPhase: Phase, phaseStatuses: Map<Phase, string>): LayoutNode[] {
  return PHASES.map((info, idx) => ({
    id: `phase-${info.id}`,
    type: 'phase' as const,
    position: { x: PHASE_X_START + idx * PHASE_X_GAP, y: PHASE_Y },
    data: {
      phaseId: info.id,
      name: info.name,
      description: info.description,
      status: derivePhaseStatus(info.id, currentPhase, phaseStatuses),
    },
  }))
}

export function buildWorkerNodes(workers: WorkerSpec[], currentPhase: Phase): LayoutNode[] {
  const grouped = new Map<Phase, WorkerSpec[]>()
  for (const w of workers) {
    const existing = grouped.get(w.phase) ?? []
    existing.push(w)
    grouped.set(w.phase, existing)
  }

  const nodes: LayoutNode[] = []

  for (const [phase, phaseWorkers] of grouped) {
    const phaseIdx = getPhaseIndex(phase)
    const phaseX = PHASE_X_START + phaseIdx * PHASE_X_GAP
    const totalWidth = (phaseWorkers.length - 1) * WORKER_X_GAP
    const startX = phaseX - totalWidth / 2

    phaseWorkers.forEach((worker, workerIdx) => {
      const isCurrentPhase = phase === currentPhase
      nodes.push({
        id: `worker-${phase}-${workerIdx}`,
        type: 'worker',
        position: { x: startX + workerIdx * WORKER_X_GAP, y: PHASE_Y + WORKER_Y_OFFSET },
        data: {
          role: worker.role,
          phase: worker.phase,
          fileOwnership: worker.fileOwnership,
          isActive: isCurrentPhase,
        },
        parentId: `phase-${phase}`,
      })
    })
  }

  return nodes
}

export function buildEdges(currentPhase: Phase): LayoutEdge[] {
  const edges: LayoutEdge[] = []
  const currentIdx = getPhaseIndex(currentPhase)

  // Forward edges: A->B->C->D->E->F->G
  for (let i = 0; i < PHASE_ORDER.length - 1; i++) {
    const source = PHASE_ORDER[i]!
    const target = PHASE_ORDER[i + 1]!
    const isCompleted = i < currentIdx
    const isActive = i === currentIdx - 1

    edges.push({
      id: `edge-${source}-${target}`,
      source: `phase-${source}`,
      target: `phase-${target}`,
      type: 'custom',
      animated: isActive,
      data: {
        active: isActive,
        completed: isCompleted,
        rollback: false,
      },
    })
  }

  // Rollback edges
  edges.push({
    id: 'edge-rollback-D-C',
    source: 'phase-D',
    target: 'phase-C',
    type: 'custom',
    data: {
      active: false,
      completed: false,
      rollback: true,
    },
  })

  edges.push({
    id: 'edge-rollback-F-E',
    source: 'phase-F',
    target: 'phase-E',
    type: 'custom',
    data: {
      active: false,
      completed: false,
      rollback: true,
    },
  })

  return edges
}
