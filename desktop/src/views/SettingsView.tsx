import { memo, useState, useCallback } from 'react'
import { Card } from '@/components/common/Card'
import { Button } from '@/components/common/Button'
import { useWorkflowStore } from '@/stores/workflow-store'
import type { ConnectionStatus } from '@/stores/workflow-store'
import { testConnection, createFlow } from '@/api/client'

const statusColors: Record<ConnectionStatus, string> = {
  disconnected: 'bg-gray-400',
  connecting: 'bg-yellow-400',
  connected: 'bg-green-400',
  error: 'bg-red-400',
}

const statusLabels: Record<ConnectionStatus, string> = {
  disconnected: 'Disconnected',
  connecting: 'Connecting...',
  connected: 'Connected',
  error: 'Error',
}

export const SettingsView = memo(function SettingsView() {
  const taskId = useWorkflowStore((s) => s.taskId)
  const apiUrl = useWorkflowStore((s) => s.apiUrl)
  const connectionStatus = useWorkflowStore((s) => s.connectionStatus)
  const setTaskId = useWorkflowStore((s) => s.setTaskId)
  const setApiUrl = useWorkflowStore((s) => s.setApiUrl)
  const setConnectionStatus = useWorkflowStore((s) => s.setConnectionStatus)

  const setFlow = useWorkflowStore((s) => s.setFlow)

  const [localApiUrl, setLocalApiUrl] = useState(apiUrl)
  const [localTaskId, setLocalTaskId] = useState(taskId ?? '')
  const [newTaskId, setNewTaskId] = useState('')
  const [newBudget, setNewBudget] = useState('10.0')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)
  const [createSuccess, setCreateSuccess] = useState(false)

  const handleConnect = useCallback(async () => {
    setConnectionStatus('connecting')
    setApiUrl(localApiUrl)
    setTaskId(localTaskId || null)

    const ok = await testConnection(localApiUrl)
    setConnectionStatus(ok ? 'connected' : 'error')
  }, [localApiUrl, localTaskId, setApiUrl, setTaskId, setConnectionStatus])

  const handleCreateFlow = useCallback(async () => {
    if (!newTaskId.trim()) return
    setCreating(true)
    setCreateError(null)
    setCreateSuccess(false)
    try {
      const flow = await createFlow(newTaskId.trim(), parseFloat(newBudget) || 10.0)
      setFlow(flow)
      setTaskId(newTaskId.trim())
      setLocalTaskId(newTaskId.trim())
      setConnectionStatus('connected')
      setCreateSuccess(true)
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : String(err))
    } finally {
      setCreating(false)
    }
  }, [newTaskId, newBudget, setFlow, setTaskId, setConnectionStatus])

  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold text-[var(--text-primary)]">Settings</h2>

      <Card header={<span className="text-sm font-medium text-[var(--text-primary)]">Create New Flow</span>}>
        <div className="space-y-4">
          <div>
            <label htmlFor="new-task-id" className="block text-xs font-medium text-[var(--text-muted)] mb-1">
              Task ID
            </label>
            <input
              id="new-task-id"
              type="text"
              value={newTaskId}
              onChange={(e) => { setNewTaskId(e.target.value); setCreateSuccess(false) }}
              placeholder="my-feature-task"
              className="w-full px-3 py-2 text-sm rounded-[8px] border border-[var(--border)] bg-[var(--bg-primary)] text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)] focus:border-transparent"
            />
          </div>
          <div>
            <label htmlFor="budget-cap" className="block text-xs font-medium text-[var(--text-muted)] mb-1">
              Budget Cap (USD)
            </label>
            <input
              id="budget-cap"
              type="number"
              min="0.01"
              step="0.5"
              value={newBudget}
              onChange={(e) => setNewBudget(e.target.value)}
              className="w-full px-3 py-2 text-sm rounded-[8px] border border-[var(--border)] bg-[var(--bg-primary)] text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)] focus:border-transparent"
            />
          </div>
          {createError && (
            <p className="text-xs text-red-400">{createError}</p>
          )}
          {createSuccess && (
            <p className="text-xs text-green-400">Flow created. Switch to Dashboard to view.</p>
          )}
          <Button size="sm" onClick={() => void handleCreateFlow()} disabled={creating || !newTaskId.trim()}>
            {creating ? 'Creating...' : 'Create Flow'}
          </Button>
        </div>
      </Card>

      <Card header={<span className="text-sm font-medium text-[var(--text-primary)]">Connection</span>}>
        <div className="space-y-4">
          <div>
            <label htmlFor="api-url" className="block text-xs font-medium text-[var(--text-muted)] mb-1">
              API URL
            </label>
            <input
              id="api-url"
              type="text"
              value={localApiUrl}
              onChange={(e) => setLocalApiUrl(e.target.value)}
              placeholder="http://localhost:9800"
              className="w-full px-3 py-2 text-sm rounded-[8px] border border-[var(--border)] bg-[var(--bg-primary)] text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)] focus:border-transparent"
            />
          </div>

          <div>
            <label htmlFor="task-id" className="block text-xs font-medium text-[var(--text-muted)] mb-1">
              Task ID
            </label>
            <input
              id="task-id"
              type="text"
              value={localTaskId}
              onChange={(e) => setLocalTaskId(e.target.value)}
              placeholder="task-001"
              className="w-full px-3 py-2 text-sm rounded-[8px] border border-[var(--border)] bg-[var(--bg-primary)] text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)] focus:border-transparent"
            />
          </div>

          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span
                className={`inline-block w-2.5 h-2.5 rounded-full ${statusColors[connectionStatus]}`}
                aria-label={`Connection status: ${statusLabels[connectionStatus]}`}
              />
              <span className="text-xs text-[var(--text-secondary)]">
                {statusLabels[connectionStatus]}
              </span>
            </div>
            <Button size="sm" onClick={() => void handleConnect()}>
              Connect
            </Button>
          </div>
        </div>
      </Card>

      <Card header={<span className="text-sm font-medium text-[var(--text-primary)]">Providers</span>}>
        <p className="text-sm text-[var(--text-muted)]">
          Provider configuration will appear here.
        </p>
      </Card>
    </div>
  )
})
