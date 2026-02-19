import { memo } from 'react'
import { Badge } from '@/components/common/Badge'

interface NodeStatusBadgeProps {
  status: 'pending' | 'active' | 'completed' | 'error'
}

const statusLabel: Record<NodeStatusBadgeProps['status'], string> = {
  pending: 'Pending',
  active: 'Active',
  completed: 'Done',
  error: 'Error',
}

export const NodeStatusBadge = memo(function NodeStatusBadge({ status }: NodeStatusBadgeProps) {
  return (
    <Badge variant={status}>
      {statusLabel[status]}
    </Badge>
  )
})
