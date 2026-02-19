import { memo } from 'react'
import { Handle, Position, type NodeProps } from '@xyflow/react'

interface WorkerNodeData extends Record<string, unknown> {
  role: string
  phase: string
  isActive: boolean
}

const providerColors: Record<string, string> = {
  integrator: '#3B82F6',
  validator: '#4CAF7D',
  coder: '#D97757',
  reviewer: '#8B5CF6',
}

export const WorkerNode = memo(function WorkerNode({ data }: NodeProps) {
  const nodeData = data as WorkerNodeData
  const color = providerColors[nodeData.role] ?? '#9A9A9E'

  return (
    <div
      className="flex items-center gap-2 px-3 py-2 bg-[var(--bg-card)] border border-[var(--border)] rounded-[12px] min-w-[110px]"
      style={{
        boxShadow: '0 1px 2px rgba(0,0,0,0.05)',
        opacity: nodeData.isActive ? 1 : 0.6,
      }}
    >
      <Handle type="target" position={Position.Top} className="!bg-[var(--border)]" />

      {/* Provider color indicator */}
      <div
        className="w-2.5 h-2.5 rounded-full shrink-0"
        style={{ backgroundColor: color }}
      />

      <div className="flex flex-col min-w-0">
        <span className="text-xs font-medium text-[var(--text-primary)] truncate">
          {String(nodeData.role)}
        </span>
        <span className="text-[10px] text-[var(--text-muted)]">
          {nodeData.isActive ? 'running' : 'done'}
        </span>
      </div>
    </div>
  )
})
