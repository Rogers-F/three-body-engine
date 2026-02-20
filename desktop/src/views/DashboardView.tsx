import { memo, useMemo, useState, useCallback } from 'react'
import { X, Play, RotateCcw, ChevronRight } from 'lucide-react'
import { WorkflowCanvas } from '@/components/workflow/WorkflowCanvas'
import { useCanvasStore } from '@/stores/canvas-store'
import { useWorkflowStore } from '@/stores/workflow-store'
import { useUIStore } from '@/stores/ui-store'
import { Card } from '@/components/common/Card'
import { Badge } from '@/components/common/Badge'
import { Button } from '@/components/common/Button'
import { mockWorkers } from '@/data/mock-workflow'
import { PHASES } from '@/types/workflow'
import { useWorkers, useFlow } from '@/api/hooks'
import { advanceFlow, createFlow } from '@/api/client'

function getDrawerTitle(
  phase: (typeof PHASES)[number] | undefined,
  worker: (typeof mockWorkers)[number] | undefined,
): string {
  if (phase) return `Phase ${phase.id}`
  if (worker) return `Worker: ${worker.role}`
  return 'Details'
}

export const DashboardView = memo(function DashboardView() {
  const selectedNodeId = useCanvasStore((s) => s.selectedNodeId)
  const selectNode = useCanvasStore((s) => s.selectNode)
  const taskId = useWorkflowStore((s) => s.taskId)
  const setTaskId = useWorkflowStore((s) => s.setTaskId)
  const setFlow = useWorkflowStore((s) => s.setFlow)
  const setConnectionStatus = useWorkflowStore((s) => s.setConnectionStatus)
  const setActiveNav = useUIStore((s) => s.setActiveNav)

  const { flow, refetch: refetchFlow } = useFlow(taskId, 3000)
  const { workers: apiWorkers, error: workersError } = useWorkers(taskId)
  const workers = taskId && !workersError && apiWorkers.length > 0 ? apiWorkers : mockWorkers

  const [actionLoading, setActionLoading] = useState(false)
  const [actionError, setActionError] = useState<string | null>(null)

  // Quick-create state
  const [quickTaskId, setQuickTaskId] = useState('')
  const [quickBudget, setQuickBudget] = useState('10.0')
  const [quickCreating, setQuickCreating] = useState(false)
  const [quickError, setQuickError] = useState<string | null>(null)

  const handleAdvance = useCallback(async () => {
    if (!taskId) return
    setActionLoading(true)
    setActionError(null)
    try {
      await advanceFlow(taskId, 'advance', 'user')
      refetchFlow()
    } catch (err) {
      setActionError(err instanceof Error ? err.message : String(err))
    } finally {
      setActionLoading(false)
    }
  }, [taskId, refetchFlow])

  const handleRollback = useCallback(async () => {
    if (!taskId) return
    setActionLoading(true)
    setActionError(null)
    try {
      await advanceFlow(taskId, 'rollback', 'user')
      refetchFlow()
    } catch (err) {
      setActionError(err instanceof Error ? err.message : String(err))
    } finally {
      setActionLoading(false)
    }
  }, [taskId, refetchFlow])

  const handleQuickCreate = useCallback(async () => {
    if (!quickTaskId.trim()) return
    setQuickCreating(true)
    setQuickError(null)
    try {
      const newFlow = await createFlow(quickTaskId.trim(), parseFloat(quickBudget) || 10.0)
      setFlow(newFlow)
      setTaskId(quickTaskId.trim())
      setConnectionStatus('connected')
    } catch (err) {
      setQuickError(err instanceof Error ? err.message : String(err))
    } finally {
      setQuickCreating(false)
    }
  }, [quickTaskId, quickBudget, setFlow, setTaskId, setConnectionStatus])

  const drawerOpen = selectedNodeId != null

  const selectedPhase = PHASES.find((p) => `phase-${p.id}` === selectedNodeId)
  const selectedWorker = workers.find(
    (w) => `worker-${w.phase}-${w.role}` === selectedNodeId,
  )
  const phaseWorkers = useMemo(
    () => selectedPhase ? workers.filter((w) => w.phase === selectedPhase.id) : [],
    [selectedPhase, workers],
  )

  const currentPhaseInfo = flow ? PHASES.find((p) => p.id === flow.currentPhase) : null

  // No task connected — show get-started prompt
  if (!taskId) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="max-w-md w-full">
          <Card header={<span className="text-sm font-medium text-[var(--text-primary)]">Get Started</span>}>
            <div className="space-y-4">
              <p className="text-sm text-[var(--text-secondary)]">
                Create a new workflow or go to Settings to connect to an existing one.
              </p>
              <div>
                <label htmlFor="quick-task-id" className="block text-xs font-medium text-[var(--text-muted)] mb-1">
                  Task ID
                </label>
                <input
                  id="quick-task-id"
                  type="text"
                  value={quickTaskId}
                  onChange={(e) => setQuickTaskId(e.target.value)}
                  placeholder="my-feature-task"
                  className="w-full px-3 py-2 text-sm rounded-[8px] border border-[var(--border)] bg-[var(--bg-primary)] text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)] focus:border-transparent"
                />
              </div>
              <div>
                <label htmlFor="quick-budget" className="block text-xs font-medium text-[var(--text-muted)] mb-1">
                  Budget Cap (USD)
                </label>
                <input
                  id="quick-budget"
                  type="number"
                  min="0.01"
                  step="0.5"
                  value={quickBudget}
                  onChange={(e) => setQuickBudget(e.target.value)}
                  className="w-full px-3 py-2 text-sm rounded-[8px] border border-[var(--border)] bg-[var(--bg-primary)] text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)] focus:border-transparent"
                />
              </div>
              {quickError && <p className="text-xs text-red-400">{quickError}</p>}
              <div className="flex gap-2">
                <Button size="sm" onClick={() => void handleQuickCreate()} disabled={quickCreating || !quickTaskId.trim()}>
                  {quickCreating ? 'Creating...' : 'Create Flow'}
                </Button>
                <Button size="sm" variant="ghost" onClick={() => setActiveNav('settings')}>
                  Settings
                  <ChevronRight size={14} className="ml-1" />
                </Button>
              </div>
            </div>
          </Card>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full gap-2">
      {/* Flow control bar */}
      <div className="flex items-center justify-between px-2 py-1.5 rounded-[8px] bg-[var(--bg-card)] border border-[var(--border)]">
        <div className="flex items-center gap-3">
          <span className="text-xs text-[var(--text-muted)]">Task:</span>
          <span className="text-sm font-mono text-[var(--text-primary)]">{taskId}</span>
          {currentPhaseInfo && (
            <>
              <span className="text-xs text-[var(--text-muted)]">Phase:</span>
              <Badge variant="active">{currentPhaseInfo.id} - {currentPhaseInfo.name}</Badge>
            </>
          )}
          {flow && (
            <>
              <span className="text-xs text-[var(--text-muted)]">Round:</span>
              <span className="text-sm text-[var(--text-secondary)]">{flow.round}</span>
              <span className="text-xs text-[var(--text-muted)]">Status:</span>
              <Badge variant={flow.status === 'running' ? 'active' : flow.status === 'completed' ? 'completed' : 'warning'}>
                {flow.status}
              </Badge>
            </>
          )}
        </div>
        <div className="flex items-center gap-2">
          {actionError && <span className="text-xs text-red-400 max-w-48 truncate">{actionError}</span>}
          <Button size="sm" variant="secondary" onClick={() => void handleRollback()} disabled={actionLoading || flow?.currentPhase === 'A'}>
            <RotateCcw size={14} className="mr-1" />
            Rollback
          </Button>
          <Button size="sm" onClick={() => void handleAdvance()} disabled={actionLoading || flow?.status === 'completed'}>
            <Play size={14} className="mr-1" />
            Advance
          </Button>
        </div>
      </div>

      {/* Canvas + Drawer */}
      <div className="flex flex-1 min-h-0 gap-4">
      <div className="flex-1 min-w-0">
        <WorkflowCanvas />
      </div>
      {drawerOpen && (
        <div className="w-80 shrink-0">
          <Card
            header={
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-[var(--text-primary)]">
                  {getDrawerTitle(selectedPhase, selectedWorker)}
                </span>
                <button
                  onClick={() => selectNode(null)}
                  className="p-1 rounded hover:bg-[var(--bg-elevated)] text-[var(--text-muted)]"
                  aria-label="Close drawer"
                >
                  <X size={16} />
                </button>
              </div>
            }
          >
            {selectedPhase && (
              <div className="space-y-3">
                <div>
                  <p className="text-xs text-[var(--text-muted)]">Name</p>
                  <p className="text-sm text-[var(--text-primary)]">{selectedPhase.name}</p>
                </div>
                <div>
                  <p className="text-xs text-[var(--text-muted)]">Description</p>
                  <p className="text-sm text-[var(--text-secondary)]">{selectedPhase.description}</p>
                </div>
                <div>
                  <p className="text-xs text-[var(--text-muted)] mb-1">Workers in phase</p>
                  <div className="flex flex-wrap gap-1">
                    {phaseWorkers.map((w) => (
                      <Badge key={w.role} variant="active">{w.role}</Badge>
                    ))}
                    {phaseWorkers.length === 0 && (
                      <span className="text-xs text-[var(--text-muted)]">None</span>
                    )}
                  </div>
                </div>
              </div>
            )}
            {selectedWorker && (
              <div className="space-y-3">
                <div>
                  <p className="text-xs text-[var(--text-muted)]">Role</p>
                  <p className="text-sm text-[var(--text-primary)]">{selectedWorker.role}</p>
                </div>
                <div>
                  <p className="text-xs text-[var(--text-muted)]">Phase</p>
                  <Badge variant="active">{selectedWorker.phase}</Badge>
                </div>
                <div>
                  <p className="text-xs text-[var(--text-muted)]">File Ownership</p>
                  <ul className="text-xs text-[var(--text-secondary)] font-mono space-y-0.5">
                    {selectedWorker.fileOwnership.map((f) => (
                      <li key={f}>{f}</li>
                    ))}
                  </ul>
                </div>
                <div>
                  <p className="text-xs text-[var(--text-muted)]">Timeouts</p>
                  <p className="text-xs text-[var(--text-secondary)]">
                    Soft: {selectedWorker.softTimeoutSec}s / Hard: {selectedWorker.hardTimeoutSec}s
                  </p>
                </div>
              </div>
            )}
            {!selectedPhase && !selectedWorker && (
              <p className="text-sm text-[var(--text-muted)]">
                Select a phase or worker node for details.
              </p>
            )}
          </Card>
        </div>
      )}
      </div>
    </div>
  )
})
