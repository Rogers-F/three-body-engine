import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  ReactFlow,
  Controls,
  Background,
  BackgroundVariant,
  type NodeMouseHandler,
  type OnNodesChange,
  applyNodeChanges,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'

import { PhaseNode } from './PhaseNode'
import { WorkerNode } from './WorkerNode'
import { PhaseEdge } from './PhaseEdge'
import { NodeActionMenu } from './NodeActionMenu'
import { buildPhaseNodes, buildWorkerNodes, buildEdges, type LayoutNode, type LayoutEdge } from './flow-layout'
import { useWorkflowStore } from '@/stores/workflow-store'
import { useCanvasStore } from '@/stores/canvas-store'
import { useUIStore } from '@/stores/ui-store'
import { mockFlowState, mockWorkers } from '@/data/mock-workflow'
import { advanceFlow } from '@/api/client'

import type { Phase } from '@/types/workflow'

const nodeTypes = {
  phase: PhaseNode,
  worker: WorkerNode,
}

const edgeTypes = {
  custom: PhaseEdge,
}

interface ContextMenu {
  x: number
  y: number
  nodeId: string
}

export function WorkflowCanvas() {
  const flow = useWorkflowStore((s) => s.flow)
  const workers = useWorkflowStore((s) => s.workers)
  const taskId = useWorkflowStore((s) => s.taskId)
  const selectNode = useCanvasStore((s) => s.selectNode)
  const openDrawer = useUIStore((s) => s.openDrawer)

  const [contextMenu, setContextMenu] = useState<ContextMenu | null>(null)

  // Use mock data when no real data is present
  const currentPhase: Phase = flow?.currentPhase ?? mockFlowState.currentPhase
  const activeWorkers = workers.length > 0 ? workers : mockWorkers

  const { nodes, edges } = useMemo(() => {
    const phaseStatuses = new Map<Phase, string>()
    const phaseNodes = buildPhaseNodes(currentPhase, phaseStatuses)

    // Count workers per phase and attach to phase nodes
    const workerCountByPhase = new Map<Phase, number>()
    for (const w of activeWorkers) {
      workerCountByPhase.set(w.phase, (workerCountByPhase.get(w.phase) ?? 0) + 1)
    }
    for (const node of phaseNodes) {
      const phaseId = node.data.phaseId as Phase
      const count = workerCountByPhase.get(phaseId)
      if (count != null) {
        node.data.workerCount = count
      }
    }

    const workerNodes = buildWorkerNodes(activeWorkers, currentPhase)
    const allEdges = buildEdges(currentPhase)

    return {
      nodes: [...phaseNodes, ...workerNodes] as LayoutNode[],
      edges: allEdges as LayoutEdge[],
    }
  }, [currentPhase, activeWorkers])

  const [localNodes, setLocalNodes] = useState(nodes)

  // Sync local state when computed nodes change
  useEffect(() => {
    setLocalNodes(nodes)
  }, [nodes])

  const onNodesChange: OnNodesChange = useCallback(
    (changes) => {
      setLocalNodes((nds) => applyNodeChanges(changes, nds) as LayoutNode[])
    },
    [],
  )

  const handleNodeClick: NodeMouseHandler = useCallback(
    (_event, node) => {
      selectNode(node.id)
      openDrawer('phase')
    },
    [selectNode, openDrawer],
  )

  const handleNodeContextMenu: NodeMouseHandler = useCallback(
    (event, node) => {
      event.preventDefault()
      setContextMenu({
        x: event.clientX,
        y: event.clientY,
        nodeId: node.id,
      })
    },
    [],
  )

  const handleContextMenuAction = useCallback(async (action: string) => {
    if (!taskId) return
    const actionMap: Record<string, string> = {
      'retry': 'rework',
      'force-pass': 'advance',
      'rollback': 'rollback',
    }
    if (action === 'details') {
      if (contextMenu) {
        selectNode(contextMenu.nodeId)
        openDrawer('phase')
      }
      return
    }
    const apiAction = actionMap[action]
    if (!apiAction) return
    try {
      await advanceFlow(taskId, apiAction, 'user')
    } catch (err) {
      console.error('Flow action failed:', err)
    }
  }, [taskId, contextMenu, selectNode, openDrawer])

  const handlePaneClick = useCallback(() => {
    setContextMenu(null)
    selectNode(null)
  }, [selectNode])

  return (
    <div className="w-full h-full" style={{ minHeight: 400 }}>
      <ReactFlow
        nodes={localNodes}
        edges={edges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        onNodesChange={onNodesChange}
        onNodeClick={handleNodeClick}
        onNodeContextMenu={handleNodeContextMenu}
        onPaneClick={handlePaneClick}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        proOptions={{ hideAttribution: true }}
        minZoom={0.3}
        maxZoom={2}
      >
        <Controls
          className="!bg-[var(--bg-card)] !border-[var(--border)] !rounded-[8px] !shadow-sm [&>button]:!bg-[var(--bg-card)] [&>button]:!border-[var(--border)] [&>button]:!text-[var(--text-secondary)] [&>button:hover]:!bg-[var(--bg-secondary)]"
        />
        <Background
          variant={BackgroundVariant.Dots}
          gap={20}
          size={1}
          color="var(--border)"
        />
      </ReactFlow>

      {contextMenu && (
        <NodeActionMenu
          x={contextMenu.x}
          y={contextMenu.y}
          nodeId={contextMenu.nodeId}
          onClose={() => setContextMenu(null)}
          onAction={handleContextMenuAction}
        />
      )}
    </div>
  )
}
