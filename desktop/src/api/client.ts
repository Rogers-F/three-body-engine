import type { FlowState, WorkerSpec, WorkflowEvent, ScoreCard, CostSummary } from '@/types/workflow'

export type { CostSummary } from '@/types/workflow'

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:9800'

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  if (!res.ok) {
    const text = await res.text().catch(() => 'Unknown error')
    throw new Error(`API error ${res.status}: ${text}`)
  }
  return res.json() as Promise<T>
}

export function getBaseUrl(): string {
  return BASE_URL
}

export async function getFlow(taskId: string): Promise<FlowState> {
  return apiFetch<FlowState>(`/api/v1/flow/${taskId}`)
}

export async function createFlow(taskId: string, budgetCapUsd: number): Promise<FlowState> {
  return apiFetch<FlowState>('/api/v1/flow', {
    method: 'POST',
    body: JSON.stringify({ task_id: taskId, budget_cap_usd: budgetCapUsd }),
  })
}

export async function advanceFlow(taskId: string, action: string, actor: string): Promise<void> {
  await apiFetch<void>(`/api/v1/flow/${taskId}/advance`, {
    method: 'POST',
    body: JSON.stringify({ action, actor }),
  })
}

export async function listWorkers(taskId: string): Promise<WorkerSpec[]> {
  return apiFetch<WorkerSpec[]>(`/api/v1/flow/${taskId}/workers`)
}

export async function listEvents(taskId: string, sinceSeq?: number): Promise<WorkflowEvent[]> {
  const query = sinceSeq != null ? `?since_seq=${sinceSeq}` : ''
  return apiFetch<WorkflowEvent[]>(`/api/v1/flow/${taskId}/events${query}`)
}

export async function listReviews(taskId: string): Promise<ScoreCard[]> {
  return apiFetch<ScoreCard[]>(`/api/v1/flow/${taskId}/reviews`)
}

export async function getCost(taskId: string): Promise<CostSummary> {
  return apiFetch<CostSummary>(`/api/v1/flow/${taskId}/cost`)
}

export async function testConnection(apiUrl: string): Promise<boolean> {
  try {
    const res = await fetch(`${apiUrl}/api/v1/health`, {
      method: 'GET',
      signal: AbortSignal.timeout(5000),
    })
    return res.ok
  } catch {
    return false
  }
}
