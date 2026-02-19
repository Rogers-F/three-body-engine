import { memo, useMemo } from 'react'
import { X } from 'lucide-react'
import { WorkflowCanvas } from '@/components/workflow/WorkflowCanvas'
import { useCanvasStore } from '@/stores/canvas-store'
import { Card } from '@/components/common/Card'
import { Badge } from '@/components/common/Badge'
import { mockWorkers } from '@/data/mock-workflow'
import { PHASES } from '@/types/workflow'

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

  const drawerOpen = selectedNodeId != null

  const selectedPhase = PHASES.find((p) => `phase-${p.id}` === selectedNodeId)
  const selectedWorker = mockWorkers.find(
    (w) => `worker-${w.phase}-${w.role}` === selectedNodeId,
  )
  const phaseWorkers = useMemo(
    () => selectedPhase ? mockWorkers.filter((w) => w.phase === selectedPhase.id) : [],
    [selectedPhase],
  )

  return (
    <div className="flex h-full gap-4">
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
  )
})
