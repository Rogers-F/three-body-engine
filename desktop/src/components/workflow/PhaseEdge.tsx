import { memo } from 'react'
import { BaseEdge, getBezierPath, type EdgeProps } from '@xyflow/react'

interface PhaseEdgeData extends Record<string, unknown> {
  active?: boolean
  completed?: boolean
  rollback?: boolean
}

export const PhaseEdge = memo(function PhaseEdge(props: EdgeProps) {
  const { sourceX, sourceY, targetX, targetY, sourcePosition, targetPosition, data } = props
  const edgeData = (data ?? {}) as PhaseEdgeData

  const isRollback = edgeData.rollback === true
  const isActive = edgeData.active === true
  const isCompleted = edgeData.completed === true

  const curvature = isRollback ? 0.5 : 0.25

  const [edgePath] = getBezierPath({
    sourceX,
    sourceY: isRollback ? sourceY + 20 : sourceY,
    targetX,
    targetY: isRollback ? targetY + 20 : targetY,
    sourcePosition,
    targetPosition,
    curvature,
  })

  let strokeColor = 'var(--border)'
  let strokeWidth = 1.5
  let strokeDasharray: string | undefined
  let animationStyle: React.CSSProperties = {}

  if (isRollback) {
    strokeColor = 'var(--phase-warning)'
    strokeDasharray = '6 4'
    strokeWidth = 1.5
  } else if (isActive) {
    strokeColor = 'var(--accent)'
    strokeWidth = 2
    strokeDasharray = '8 4'
    animationStyle = {
      animation: 'edge-flow 1s linear infinite',
    }
  } else if (isCompleted) {
    strokeColor = 'var(--phase-completed)'
    strokeWidth = 2
  }

  return (
    <BaseEdge
      path={edgePath}
      style={{
        stroke: strokeColor,
        strokeWidth,
        strokeDasharray,
        fill: 'none',
        ...animationStyle,
      }}
    />
  )
})
