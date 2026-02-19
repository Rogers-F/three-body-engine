/** Phase identifiers A through G */
export type Phase = 'A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G'

/** Overall flow lifecycle status (matches backend domain.FlowStatus) */
export type FlowStatus = 'running' | 'blocked' | 'failed' | 'completed'

/** Provider identifiers for the three-body system */
export type Provider = 'claude' | 'codex' | 'gemini'

/** Cost governor action */
export type CostActionType = 'continue' | 'warn' | 'halt'

/** State of the overall workflow (matches backend domain.FlowState) */
export interface FlowState {
  taskId: string
  currentPhase: Phase
  status: FlowStatus
  stateVersion: number
  round: number
  budgetUsedUsd: number
  budgetCapUsd: number
  lastEventSeq: number
  updatedAtUnix: number
}

/** Transition trigger for phase advancement (matches backend domain.TransitionTrigger) */
export interface TransitionTrigger {
  action: string // advance / rollback / rework
  actor: string  // lead / system / user
  payload?: unknown
}

/** Gate evaluation result (matches backend domain.GateDecision) */
export interface GateDecision {
  allow: boolean
  blockers: string[]
  retryable: boolean
  nextPhase: Phase
  requireOps: string[]
}

/** Worker lifecycle state (matches backend domain.WorkerState) */
export type WorkerStatus = 'created' | 'running' | 'soft_timeout' | 'hard_timeout' | 'replaced' | 'done'

/** Worker specification (matches backend domain.WorkerSpec) */
export interface WorkerSpec {
  taskId: string
  phase: Phase
  role: string
  fileOwnership: string[]
  digestPath: string
  softTimeoutSec: number
  hardTimeoutSec: number
}

/** Intent for file operations (matches backend domain.Intent) */
export interface Intent {
  intentId: string
  taskId: string
  workerId: string
  targetFile: string
  operation: string
  status: string
  preHash: string
  postHash: string
  payloadHash: string
  leaseUntil: number
}

/** Reference to a versioned artifact (matches backend domain.ArtifactRef) */
export interface ArtifactRef {
  id: string
  type: string
  path: string
  version: number
  hash: string
}

/** Deadline with soft and hard limits */
export interface Deadline {
  soft: string
  hard: string
}

/** Compact context digest passed to workers (matches backend domain.ContextDigest) */
export interface ContextDigest {
  taskId: string
  phaseId: string
  objective: string
  constraints: string[]
  fileOwnership: string[]
  deadline: Deadline
  artifactRefs: ArtifactRef[]
  codingStandards: string
}

/** Compaction slots — 9 semantic slots preserved across phases (matches backend domain.CompactionSlots) */
export interface CompactionSlots {
  taskSpec: string
  acceptanceCriteria: string
  currentPhase: string
  openRisks: string[]
  activeConstraints: string[]
  fileOwnership: string[]
  artifactRefs: ArtifactRef[]
  pendingIntents: string[]
  nextPhaseReqs: string[]
}

/** Workflow event (matches backend domain.WorkflowEvent) */
export interface WorkflowEvent {
  id: number
  taskId: string
  seqNo: number
  phase: Phase
  eventType: string
  payloadJson: string
  createdAt: number
}

/** Phase snapshot at boundary (matches backend domain.PhaseSnapshot) */
export interface PhaseSnapshot {
  id: number
  taskId: string
  phase: Phase
  round: number
  snapshotJson: string
  checksum: string
  createdAt: number
}

/** Audit record (matches backend domain.AuditRecord) */
export interface AuditRecord {
  id: string
  taskId: string
  category: string // guard / review / permission / config / bridge
  actor: string    // system / lead / worker / codex / gemini
  action: string
  requestJson: string
  decisionJson: string
  severity: string
  createdAt: number
}

/** Review scores — 5 dimensions, 1-5 each (matches backend domain.Scores) */
export interface Scores {
  correctness: number
  security: number
  maintainability: number
  cost: number
  deliveryRisk: number
}

/** Issue found during review (matches backend domain.Issue) */
export interface Issue {
  severity: 'P0' | 'P1' | 'P2'
  location: string
  description: string
  suggestion: string
  evidence: string
}

/** Structured review output (matches backend domain.ScoreCard) */
export interface ScoreCard {
  reviewId: string
  reviewer: string
  scores: Scores
  issues: Issue[]
  alternatives: string[]
  verdict: 'pass' | 'conditional_pass' | 'fail'
}

/** Aggregated review decision (matches backend domain.ConsensusResult) */
export interface ConsensusResult {
  weightedScore: number
  blocking: boolean
  blockReasons: string[]
  finalVerdict: string
}

/** Session config for a code agent (matches backend domain.SessionConfig) */
export interface SessionConfig {
  taskId: string
  role: string
  workspace: string
  env: Record<string, string>
  timeoutSec: number
  contextFile: string
}

/** Normalized event from any provider (matches backend domain.NormalizedEvent) */
export interface NormalizedEvent {
  type: string
  provider: Provider
  sessionId: string
  payload: unknown
}

/** Cost tracking delta (matches backend domain.CostDelta) */
export interface CostDelta {
  inputTokens: number
  outputTokens: number
  amountUsd: number
  provider: Provider
  phase: Phase
}

/** Phase metadata for display */
export interface PhaseInfo {
  id: Phase
  name: string
  description: string
}

export const PHASES: PhaseInfo[] = [
  { id: 'A', name: 'Understand', description: 'Task analysis & user confirmation' },
  { id: 'B', name: 'Strategize', description: 'Architecture & compaction' },
  { id: 'C', name: 'Execute', description: 'Code generation & testing' },
  { id: 'D', name: 'Review', description: 'Multi-provider code review' },
  { id: 'E', name: 'Integrate', description: 'Merge & conflict resolution' },
  { id: 'F', name: 'Validate', description: 'Final validation & approval' },
  { id: 'G', name: 'Complete', description: 'Delivery & cleanup' },
]
