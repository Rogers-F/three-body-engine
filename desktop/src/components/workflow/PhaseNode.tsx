import { memo } from 'react'
import { Handle, Position, type NodeProps } from '@xyflow/react'
import { NodeStatusBadge } from './NodeStatusBadge'
import { useUIStore } from '@/stores/ui-store'
import { useCanvasStore } from '@/stores/canvas-store'

type PhaseStatus = 'pending' | 'active' | 'completed' | 'error'

interface PhaseNodeData extends Record<string, unknown> {
  phaseId: string
  name: string
  description: string
  status: PhaseStatus
  workerCount?: number
}

const statusStripeColor: Record<PhaseStatus, string> = {
  pending: 'var(--phase-pending)',
  active: 'var(--phase-active)',
  completed: 'var(--phase-completed)',
  error: 'var(--phase-error)',
}

export const PhaseNode = memo(function PhaseNode({ id, data }: NodeProps) {
  const nodeData = data as PhaseNodeData
  const openDrawer = useUIStore((s) => s.openDrawer)
  const selectNode = useCanvasStore((s) => s.selectNode)

  const { status } = nodeData
  const isActive = status === 'active'

  const handleClick = () => {
    selectNode(id)
    openDrawer('phase')
  }

  return (
    <div
      onClick={handleClick}
      className="relative flex cursor-pointer"
      style={{
        animation: isActive ? 'pulse-glow 2s ease-in-out infinite' : undefined,
      }}
      role="button"
      tabIndex={0}
      aria-label={`Phase ${String(nodeData.phaseId)}: ${String(nodeData.name)}`}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          handleClick()
        }
      }}
    >
      <Handle type="target" position={Position.Left} className="!bg-[var(--border)]" />

      {/* Left color stripe */}
      <div
        className="w-1 rounded-l-[16px] shrink-0"
        style={{ backgroundColor: statusStripeColor[status] }}
      />

      {/* Card body */}
      <div
        className="flex flex-col gap-1.5 px-3 py-2.5 bg-[var(--bg-card)] border border-[var(--border)] rounded-r-[16px] min-w-[140px]"
        style={{
          borderLeft: 'none',
          boxShadow: '0 1px 3px rgba(0,0,0,0.08), 0 1px 2px rgba(0,0,0,0.04)',
        }}
      >
        <div className="flex items-center gap-2">
          <span className="text-lg font-bold text-[var(--text-primary)]">
            {String(nodeData.phaseId)}
          </span>
          <span className="text-sm font-medium text-[var(--text-primary)]">
            {String(nodeData.name)}
          </span>
        </div>

        <div className="flex items-center gap-2">
          <NodeStatusBadge status={status} />
          {nodeData.workerCount != null && Number(nodeData.workerCount) > 0 && (
            <span className="text-xs text-[var(--text-muted)]">
              {String(nodeData.workerCount)} worker{Number(nodeData.workerCount) !== 1 ? 's' : ''}
            </span>
          )}
        </div>
      </div>

      <Handle type="source" position={Position.Right} className="!bg-[var(--border)]" />
    </div>
  )
})
