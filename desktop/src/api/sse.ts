import type { WorkflowEvent } from '@/types/workflow'

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:9800'

export function createEventSource(
  taskId: string,
  onEvent: (event: WorkflowEvent) => void,
  onError?: (error: Event) => void,
): EventSource {
  const url = `${BASE_URL}/api/v1/flow/${taskId}/events/stream`
  const source = new EventSource(url)

  source.onmessage = (msg) => {
    try {
      const parsed = JSON.parse(msg.data as string) as WorkflowEvent
      onEvent(parsed)
    } catch {
      // Ignore malformed events
    }
  }

  if (onError) {
    source.onerror = onError
  }

  return source
}

export function closeEventSource(source: EventSource): void {
  source.close()
}
