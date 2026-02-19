import { useState, useEffect, useCallback, useRef } from 'react'
import type { FlowState, WorkerSpec, WorkflowEvent, ScoreCard, CostSummary } from '@/types/workflow'
import { getFlow, listWorkers, listEvents, listReviews, getCost } from './client'
import { createEventSource, closeEventSource } from './sse'

interface AsyncState<T> {
  data: T
  loading: boolean
  error: Error | null
}

export function useFlow(
  taskId: string | null,
  pollIntervalMs?: number,
): { flow: FlowState | null; loading: boolean; error: Error | null; refetch: () => void } {
  const [state, setState] = useState<AsyncState<FlowState | null>>({
    data: null,
    loading: false,
    error: null,
  })

  const fetchFlow = useCallback(async () => {
    if (!taskId) return
    setState((prev) => ({ ...prev, loading: true, error: null }))
    try {
      const flow = await getFlow(taskId)
      setState({ data: flow, loading: false, error: null })
    } catch (err) {
      setState((prev) => ({ ...prev, loading: false, error: err instanceof Error ? err : new Error(String(err)) }))
    }
  }, [taskId])

  useEffect(() => {
    if (!taskId) {
      setState({ data: null, loading: false, error: null })
      return
    }

    void fetchFlow()

    if (pollIntervalMs && pollIntervalMs > 0) {
      const id = setInterval(() => void fetchFlow(), pollIntervalMs)
      return () => clearInterval(id)
    }
  }, [taskId, pollIntervalMs, fetchFlow])

  return { flow: state.data, loading: state.loading, error: state.error, refetch: fetchFlow }
}

export function useWorkers(
  taskId: string | null,
): { workers: WorkerSpec[]; loading: boolean; error: Error | null } {
  const [state, setState] = useState<AsyncState<WorkerSpec[]>>({
    data: [],
    loading: false,
    error: null,
  })

  useEffect(() => {
    if (!taskId) {
      setState({ data: [], loading: false, error: null })
      return
    }

    let cancelled = false
    setState((prev) => ({ ...prev, loading: true, error: null }))

    void listWorkers(taskId).then(
      (workers) => { if (!cancelled) setState({ data: workers, loading: false, error: null }) },
      (err: unknown) => { if (!cancelled) setState((prev) => ({ ...prev, loading: false, error: err instanceof Error ? err : new Error(String(err)) })) },
    )

    return () => { cancelled = true }
  }, [taskId])

  return { workers: state.data, loading: state.loading, error: state.error }
}

export function useEvents(
  taskId: string | null,
  streaming?: boolean,
): { events: WorkflowEvent[]; loading: boolean; error: Error | null } {
  const [state, setState] = useState<AsyncState<WorkflowEvent[]>>({
    data: [],
    loading: false,
    error: null,
  })
  const sourceRef = useRef<EventSource | null>(null)

  useEffect(() => {
    if (!taskId) {
      setState({ data: [], loading: false, error: null })
      return
    }

    let cancelled = false
    setState((prev) => ({ ...prev, loading: true, error: null }))

    void listEvents(taskId).then(
      (events) => { if (!cancelled) setState({ data: events, loading: false, error: null }) },
      (err: unknown) => { if (!cancelled) setState((prev) => ({ ...prev, loading: false, error: err instanceof Error ? err : new Error(String(err)) })) },
    )

    if (streaming) {
      const source = createEventSource(
        taskId,
        (event) => {
          if (!cancelled) {
            setState((prev) => ({ ...prev, data: [...prev.data, event] }))
          }
        },
        () => {
          if (!cancelled) {
            setState((prev) => ({ ...prev, error: new Error('SSE connection error') }))
          }
        },
      )
      sourceRef.current = source
    }

    return () => {
      cancelled = true
      if (sourceRef.current) {
        closeEventSource(sourceRef.current)
        sourceRef.current = null
      }
    }
  }, [taskId, streaming])

  return { events: state.data, loading: state.loading, error: state.error }
}

export function useCost(
  taskId: string | null,
): { cost: CostSummary | null; loading: boolean; error: Error | null } {
  const [state, setState] = useState<AsyncState<CostSummary | null>>({
    data: null,
    loading: false,
    error: null,
  })

  useEffect(() => {
    if (!taskId) {
      setState({ data: null, loading: false, error: null })
      return
    }

    let cancelled = false
    setState((prev) => ({ ...prev, loading: true, error: null }))

    void getCost(taskId).then(
      (cost) => { if (!cancelled) setState({ data: cost, loading: false, error: null }) },
      (err: unknown) => { if (!cancelled) setState((prev) => ({ ...prev, loading: false, error: err instanceof Error ? err : new Error(String(err)) })) },
    )

    return () => { cancelled = true }
  }, [taskId])

  return { cost: state.data, loading: state.loading, error: state.error }
}

export function useReviews(
  taskId: string | null,
): { reviews: ScoreCard[]; loading: boolean; error: Error | null } {
  const [state, setState] = useState<AsyncState<ScoreCard[]>>({
    data: [],
    loading: false,
    error: null,
  })

  useEffect(() => {
    if (!taskId) {
      setState({ data: [], loading: false, error: null })
      return
    }

    let cancelled = false
    setState((prev) => ({ ...prev, loading: true, error: null }))

    void listReviews(taskId).then(
      (reviews) => { if (!cancelled) setState({ data: reviews, loading: false, error: null }) },
      (err: unknown) => { if (!cancelled) setState((prev) => ({ ...prev, loading: false, error: err instanceof Error ? err : new Error(String(err)) })) },
    )

    return () => { cancelled = true }
  }, [taskId])

  return { reviews: state.data, loading: state.loading, error: state.error }
}
